package main

import (
	"context"
	"database/sql"
	"livon/internal/app/registry"
	"livon/internal/app/server"
	"livon/internal/app/worker"
	"livon/internal/config"
	"livon/internal/core/services"
	"livon/internal/platform/logger"
	"livon/internal/platform/telemetry"
	"livon/internal/plugins/postgres"
	redisPlugin "livon/internal/plugins/redis"
	"livon/internal/plugins/twilio"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Config
	cfg := config.Load()

	// Logger
	log := logger.NewLogger(*cfg)
	log.Info("starting application")

	otelShutdown, err := telemetry.InitTelemetry(ctx, *cfg)
	if err != nil {
		log.Error("failed to initialize telemetry", "err", err)
	}
	defer func() {
		log.Info("flushing telemetry...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelShutdown(shutdownCtx); err != nil {
			log.Error("telemetry shutdown failed", "err", err)
		}
	}()

	// Infra
	var pdb *sql.DB
	if pdb, err = postgres.New(ctx, *cfg.Postgres); err != nil {
		log.Error("postgress connection failed", "DSN", cfg.Postgres.DSN)
		return
	}
	log.Info("postgress connected")
	var rdb *redis.Client
	if rdb, err = redisPlugin.NewRedisClient(ctx, *cfg.Redis); err != nil {
		log.Error("redis connection failed", "url", cfg.Redis.URL)
		return
	}
	log.Info("redis connected")
	// Adapters
	userRepo := postgres.NewUserRepository(pdb)
	convRepo := postgres.NewConversationRepo(pdb)
	partRepo := postgres.NewParticipantRepo(pdb)
	msgRepo := postgres.NewMessageRepo(pdb)
	presStore := redisPlugin.NewRedisPresenceStore(rdb)
	msgQueue := redisPlugin.NewRedisMessageQueue(rdb)

	tw := twilio.NewTwilioClient(*cfg.Twilio)

	// Core Services
	hub := registry.NewRegistry()
	txManager := services.NewTxManager(log, pdb)
	userSvc := services.NewUserService(log, userRepo, tw)
	sessSvc := services.NewSessionService(log, partRepo, txManager)
	msgSvc := services.NewMessageService(log, msgQueue, hub, msgRepo, txManager)

	tokenSvc := services.NewTokenService(log, cfg.SecretToken)
	managerSvc := services.NewManagerService(log, convRepo, presStore, sessSvc, msgSvc, txManager)

	wrkr := worker.NewConversationWorker(log, *msgQueue, msgSvc, cfg.Worker.MessageGroup)
	hub.RunWorker(wrkr.Run)

	// Server
	srv := server.NewServer(log, cfg.Service.Name, "8080", userSvc, tokenSvc, managerSvc, hub)
	srv.Start()
}
