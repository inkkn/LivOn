package server

import (
	"log/slog"
	"net/http"
	"time"

	"livon/internal/app/registry"
	"livon/internal/app/server/handlers"
	"livon/internal/core/services"
	"livon/pkg/middleware"
)

type Server struct {
	log         *slog.Logger
	mux         *http.ServeMux
	port        string
	authHandler *handlers.AuthHandler
	wsHandler   *handlers.WSHandler
	tokenSvc    *services.TokenService
}

func NewServer(
	log *slog.Logger,
	port string,
	userSvc *services.UserService,
	tokenSvc *services.TokenService,
	managerSvc *services.ManagerService,
	hub *registry.Registry,
) *Server {
	s := &Server{
		log:         log,
		mux:         http.NewServeMux(),
		port:        port,
		authHandler: handlers.NewAuthHandler(userSvc, tokenSvc),
		wsHandler:   handlers.NewWSHandler(hub, managerSvc),
		tokenSvc:    tokenSvc,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	// Initialize Middleware
	auth := middleware.AuthMiddleware(s.tokenSvc)
	log := middleware.RequestLogger(s.log)
	// Public Routes
	s.mux.HandleFunc("POST /auth/register", s.authHandler.RequestOTP)
	s.mux.HandleFunc("POST /auth/verify", s.authHandler.VerifyOTP)

	// Protected Routes
	// The middleware extracts the 'sub' (phone) from JWT and puts it in Context.
	s.mux.Handle("/ws", log(auth(http.HandlerFunc(s.wsHandler.Handler))))
}

func (s *Server) Start() error {
	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	s.log.Info("starting server", "port", s.port)
	return server.ListenAndServe()
}
