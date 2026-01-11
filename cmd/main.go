package main

import (
	"context"
	"database/sql"
	"livon/internal/app/registry"
	"livon/internal/app/server"
	"livon/internal/app/worker"
	"livon/internal/config"
	"livon/internal/core/services"
	"livon/internal/plugins/postgres"
	redisPlugin "livon/internal/plugins/redis"
	"livon/internal/plugins/twilio"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Context
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Config
	cfg := config.Load()

	// Infra
	var pdb *sql.DB
	var err error
	if pdb, err = postgres.New(ctx, *cfg.Postgres); err != nil {
		return
	}
	var rdb *redis.Client
	if rdb, err = redisPlugin.NewRedisClient(ctx, *cfg.Redis); err != nil {
		return
	}

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
	txManager := services.NewTxManager(pdb)
	userSvc := services.NewUserService(userRepo, tw)
	sessSvc := services.NewSessionService(partRepo, txManager)
	msgSvc := services.NewMessageService(msgQueue, hub, msgRepo, txManager)

	tokenSvc := services.NewTokenService(cfg.SecretToken)
	managerSvc := services.NewManagerService(convRepo, presStore, sessSvc, msgSvc)

	wrkr := worker.NewConversationWorker(*msgQueue, msgSvc, cfg.Worker.MessageGroup)
	hub.RunWorker(wrkr.Run)

	// Server
	srv := server.NewServer("8080", userSvc, tokenSvc, managerSvc, hub)
	srv.Start()
}
