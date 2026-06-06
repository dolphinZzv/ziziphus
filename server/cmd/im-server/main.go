package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dolphinz/im-server/config"
	"github.com/dolphinz/im-server/internal/api"
	"github.com/dolphinz/im-server/internal/auth"
	"github.com/dolphinz/im-server/internal/conversation"
	"github.com/dolphinz/im-server/internal/gateway"
	"github.com/dolphinz/im-server/internal/handler"
	"github.com/dolphinz/im-server/internal/message"
	"github.com/dolphinz/im-server/internal/session"
	"github.com/dolphinz/im-server/internal/storage/cache"
	"github.com/dolphinz/im-server/internal/storage/db"
	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
)

func main() {
	cfg, err := config.Load("config/config.yaml")
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

	// Crypto (RSA for passwords)
	var crypto *auth.Crypto
	if cfg.JWT.PrivateKeyPath != "" {
		crypto, err = auth.NewCrypto(cfg.JWT.PrivateKeyPath)
		if err != nil {
			logger.Error("load rsa key failed", "error", err)
			os.Exit(1)
		}
	} else {
		// Generate in-memory key for dev
		priv, err := rsa.GenerateKey(rand.Reader, 2048)
		if err != nil {
			logger.Error("generate rsa key failed", "error", err)
			os.Exit(1)
		}
		crypto = auth.NewCryptoFromKeys(priv)
	}

	// Auth service
	authSvc := auth.NewService(crypto, cfg.JWT.Secret, cfg.JWT.ExpireHours, userRepo)

	// Session manager
	sessMgr := session.NewManager(sessCache, sessRepo)

	// Conversation manager
	convMgr := conversation.NewManager(convRepo, msgRepo, seqCache)

	// Gateway
	gwMgr := gateway.NewManager()

	// Rate limiter
	rl := message.NewRateLimiter(cfg.RateLimit.MsgPerSec, cfg.RateLimit.BurstSize, cfg.RateLimit.MaxBodyBytes)

	// Message router
	msgRouter := message.NewRouter(sessMgr, convMgr, gwMgr)

	// Pusher
	pusher := message.NewPusher(gwMgr, receiptRepo)

	// Ingest pipeline
	ingest := message.NewIngest(msgRepo, msgRouter, pusher, rl, sf, seqCache, convMgr)

	// Sync handler
	syncHandler := message.NewSyncHandler(msgRepo, seqCache)

	// Receipt handler
	readReceiptRepo := receiptRepo // satisfies receiptWriter interface
	receiptHandler := message.NewReceiptHandler(msgRepo, seqCache, convRepo, gwMgr, readReceiptRepo)

	// HTTP API handlers
	userHandler := api.NewUserHandler(authSvc, userRepo, sessMgr)
	convHandler := api.NewConvHandler(convMgr, convRepo, seqCache, receiptHandler, ingest, sf.NextID)
	msgHandler := api.NewMsgHandler(msgRepo)
	contactHandler := api.NewContactHandler(contactRepo, userRepo, sessMgr)
	handlers := &api.Handlers{
		User:         userHandler,
		Conversation: convHandler,
		Message:      msgHandler,
		Contact:      contactHandler,
	}

	// Auth middleware
	authMW := auth.AuthMiddleware(authSvc)
	wsAuthMW := auth.WSAuthMiddleware(authSvc)

	// WS handler
	wsHandler := handler.NewWSHandler(wsAuthMW, sessMgr, gwMgr, ingest, syncHandler, receiptHandler)

	// Router
	r := api.NewRouter(handlers, authMW)
	r.Handle("/ws", wsHandler)

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
