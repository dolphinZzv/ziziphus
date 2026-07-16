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
	"ziziphus/config"
	"ziziphus/internal/api"
	"ziziphus/internal/auth"
	"ziziphus/internal/conversation"
	"ziziphus/internal/gateway"
	"ziziphus/internal/handler"
	"ziziphus/internal/healthcheck"
	"ziziphus/internal/message"
	"ziziphus/internal/session"
	"ziziphus/internal/storage/cache"
	"ziziphus/internal/storage/db"
	"ziziphus/internal/storage/file"
	"ziziphus/internal/webembed"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"

	_ "ziziphus/docs" // swagger docs
)

//	@title			Ziziphus API
//	@version		1.0
//	@description	Ziziphus IM 服务 REST API 文档
//	@host			localhost:8080
//	@BasePath		/api/v1

//	@securityDefinitions.apikey	Bearer
//	@in							header
//	@name						Authorization
//	@description				Bearer token from login/register response

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

	// Startup dependency check
	if err := healthcheck.Check(ctx, pool, rdb); err != nil {
		logger.Error("dependency check failed", "error", err)
		os.Exit(1)
	}

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

	// Webhook repo (used by both ingest forwarding and webhook handler)
	webhookRepo := db.NewWebhookRepo(pool)

	// Ingest pipeline
	contactReqRepo := db.NewContactRequestRepo(pool)
	ingest := message.NewIngest(msgRepo, msgRouter, pusher, rl, sf, seqCache, convMgr, contactReqRepo, contactRepo, userRepo, webhookRepo)

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
	userHandler := api.NewUserHandler(authSvc, userRepo, sessMgr, sf.NextID, mfaRepo, emailVerifyRepo, mailer, cfg.Server.RegistrationAllowed())
	convHandler := api.NewConvHandler(convMgr, convRepo, seqCache, receiptHandler, ingest, userRepo, sf.NextID)
	msgHandler := api.NewMsgHandler(msgRepo, receiptRepo, convMgr)
	contactHandler := api.NewContactHandler(contactRepo, contactReqRepo, userRepo, sessMgr, ingest, convMgr)
	sessionHandler := api.NewSessionHandler(sessMgr, gwMgr)
	webhookHandler := api.NewWebhookHandler(webhookRepo, sf, convMgr, userRepo, msgRepo, msgRouter, pusher, seqCache, ingest)
	handlers := &api.Handlers{
		User:         userHandler,
		Conversation: convHandler,
		Message:      msgHandler,
		Contact:      contactHandler,
		Session:      sessionHandler,
		File:         fileHandler,
		Webhook:      webhookHandler,
		Announcement: api.Announcement(cfg.Announcement),
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
	_ = srv.Shutdown(shutdownCtx)
}
