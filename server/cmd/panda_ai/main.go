package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/afero"
	"siciv.space/agent/panda_ai/config"
	"siciv.space/agent/panda_ai/internal/api"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/conversation"
	"siciv.space/agent/panda_ai/internal/gateway"
	"siciv.space/agent/panda_ai/internal/handler"
	"siciv.space/agent/panda_ai/internal/message"
	"siciv.space/agent/panda_ai/internal/session"
	"siciv.space/agent/panda_ai/internal/storage/cache"
	"siciv.space/agent/panda_ai/internal/storage/db"
	"siciv.space/agent/panda_ai/internal/storage/file"
	"siciv.space/agent/panda_ai/internal/webembed"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

func main() {
	configPath := flag.String("c", "config/config.yaml", "config file path")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}
	logger.SetLevel(slog.LevelInfo)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Database
	pool, err := db.NewPgPool(ctx, cfg.Postgres.DSN, cfg.Postgres.MaxConns)
	if err != nil {
		logger.Error("db init failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := db.RunMigrations(ctx, pool, cfg.Postgres.Migrations); err != nil {
		logger.Error("migrations failed", "error", err)
		os.Exit(1)
	}

	// Redis
	rdb, err := cache.NewRedisClient(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	if err != nil {
		logger.Error("redis init failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// Repos
	userRepo := db.NewUserRepo(pool)
	sessRepo := db.NewSessionRepo(pool)
	convRepo := db.NewConvRepo(pool)
	msgRepo := db.NewMessageRepo(pool)
	contactRepo := db.NewContactRepo(pool)
	receiptRepo := db.NewReceiptRepo(pool)
	joinRequestRepo := db.NewJoinRequestRepo(pool)
	fileRepo := db.NewFileRepo(pool)
	mfaRepo := db.NewMFARepo(pool)
	emailVerifyRepo := db.NewEmailVerifyRepo(pool)
	mailer := auth.NewMailer(cfg.SMTP)

	// Caches
	sessCache := cache.NewSessionCache(rdb)
	seqCache := cache.NewSeqCache(rdb)

	// Snowflake
	startTime, err := time.Parse(time.RFC3339, cfg.Snowflake.StartTime)
	if err != nil {
		logger.Error("invalid snowflake start_time", "error", err)
		os.Exit(1)
	}
	sf := model.NewSnowflake(cfg.Snowflake.WorkerID, startTime)

	// Auth service (bcrypt password hashing + JWT with refresh tokens)
	authSvc := auth.NewService(cfg.JWT.Secret, cfg.JWT.ExpireHours, cfg.JWT.RefreshExpireHours, userRepo, rdb, sf.NextID)

	// Session manager
	sessMgr := session.NewManager(sessCache, sessRepo)

	// Conversation manager
	convMgr := conversation.NewManager(convRepo, msgRepo, seqCache, userRepo, joinRequestRepo)

	// Gateway
	gwMgr := gateway.NewManager()

	// Rate limiter
	rl := message.NewRateLimiter(cfg.RateLimit.MsgPerSec, cfg.RateLimit.BurstSize, cfg.RateLimit.MaxBodyBytes)

	// Message router
	msgRouter := message.NewRouter(sessMgr, convMgr, gwMgr)

	// Pusher
	pusher := message.NewPusher(gwMgr, receiptRepo)

	// Ingest pipeline
	contactReqRepo := db.NewContactRequestRepo(pool)
	ingest := message.NewIngest(msgRepo, msgRouter, pusher, rl, sf, seqCache, convMgr, contactReqRepo, contactRepo, userRepo)

	// Sync handler
	syncHandler := message.NewSyncHandler(msgRepo, seqCache)

	// Receipt handler
	readReceiptRepo := receiptRepo // satisfies receiptWriter interface
	receiptHandler := message.NewReceiptHandler(msgRepo, seqCache, convRepo, gwMgr, readReceiptRepo)

	// File storage (using afero for filesystem abstraction)
	fileFs := afero.NewOsFs()
	fileStore := file.NewStore(fileFs, cfg.Storage.LocalPath)
	fileHandler := api.NewFileHandler(fileStore, fileRepo, sf, cfg.Storage.BaseURL, convMgr, ingest, userRepo)

	// HTTP API handlers
	userHandler := api.NewUserHandler(authSvc, userRepo, sessMgr, sf.NextID, mfaRepo, emailVerifyRepo, mailer)
	convHandler := api.NewConvHandler(convMgr, convRepo, seqCache, receiptHandler, ingest, userRepo, sf.NextID)
	msgHandler := api.NewMsgHandler(msgRepo, receiptRepo, convMgr)
	contactHandler := api.NewContactHandler(contactRepo, contactReqRepo, userRepo, sessMgr, ingest, convMgr)
	sessionHandler := api.NewSessionHandler(sessMgr, gwMgr)
	handlers := &api.Handlers{
		User:         userHandler,
		Conversation: convHandler,
		Message:      msgHandler,
		Contact:      contactHandler,
		Session:      sessionHandler,
		File:         fileHandler,
		DB:           pool,
		RDB:          rdb,
	}

	// Auth middleware
	authMW := auth.AuthMiddlewareWithAPIKey(authSvc, userRepo)
	wsAuthMW := auth.WSAuthMiddleware(authSvc, userRepo)

	// WS handler
	wsHandler := handler.NewWSHandler(wsAuthMW, sessMgr, gwMgr, ingest, syncHandler, receiptHandler, msgRepo)

	// Router
	r := api.NewRouter(handlers, authMW)
	r.Handle("/ws", wsHandler)

	// SPA fallback for embedded web frontend
	r.NotFound(webembed.Handler().ServeHTTP)

	// Heartbeat
	hbCfg := gateway.DefaultHeartbeatConfig()
	hb := gateway.NewHeartbeat(gwMgr, hbCfg)
	go hb.Start(ctx, func(ctx context.Context, connID string) {
		gwMgr.Remove(ctx, connID)
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("server starting", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	logger.Info("shutting down...")

	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	srv.Shutdown(shutdownCtx)
}
