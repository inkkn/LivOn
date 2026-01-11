package server

import (
	"log"
	"net/http"
	"time"

	"livon/internal/app/registry"
	"livon/internal/app/server/handlers"
	"livon/internal/core/services"
	"livon/pkg/middleware"
)

type Server struct {
	mux         *http.ServeMux
	port        string
	authHandler *handlers.AuthHandler
	wsHandler   *handlers.WSHandler
	tokenSvc    *services.TokenService
}

func NewServer(
	port string,
	userSvc *services.UserService,
	tokenSvc *services.TokenService,
	managerSvc *services.ManagerService,
	hub *registry.Registry,
) *Server {
	s := &Server{
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
	// 1. Initialize Middleware
	auth := middleware.AuthMiddleware(s.tokenSvc)

	// 2. Public Routes
	s.mux.HandleFunc("POST /auth/register", s.authHandler.RequestOTP)
	s.mux.HandleFunc("POST /auth/verify", s.authHandler.VerifyOTP)

	// 3. Protected Routes
	// We wrap the WSHandler with the Auth middleware.
	// The middleware extracts the 'sub' (phone) from JWT and puts it in Context.
	s.mux.Handle("/ws", auth(http.HandlerFunc(s.wsHandler.Handler)))
}

func (s *Server) Start() error {
	server := &http.Server{
		Addr:         ":" + s.port,
		Handler:      s.mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	log.Printf("Server starting on port %s", s.port)
	return server.ListenAndServe()
}
