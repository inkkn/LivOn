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
	app         string
	port        string
	authHandler *handlers.AuthHandler
	wsHandler   *handlers.WSHandler
	tokenSvc    *services.TokenService
}

func NewServer(
	log *slog.Logger,
	app string,
	port string,
	userSvc *services.UserService,
	tokenSvc *services.TokenService,
	managerSvc *services.ManagerService,
	hub *registry.Registry,
) *Server {
	s := &Server{
		app:         app,
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
	trace := middleware.TracerMiddleware(s.app)
	// Public Routes
	s.mux.Handle("POST /auth/register", trace(log(http.Handler(http.HandlerFunc(s.authHandler.RequestOTP)))))
	s.mux.Handle("POST /auth/verify", trace(log(http.Handler(http.HandlerFunc(s.authHandler.VerifyOTP)))))

	// Protected Routes
	// The middleware extracts the 'sub' (phone) from JWT and puts it in Context.
	s.mux.Handle("/ws", trace(log(auth(http.HandlerFunc(s.wsHandler.Handler)))))
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
