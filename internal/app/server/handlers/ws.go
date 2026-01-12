package handlers

import (
	"context"
	"livon/internal/app/registry"
	"livon/internal/app/server/ws"
	"livon/internal/core/domain"
	"livon/internal/core/services"
	"livon/pkg/middleware"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

type WSHandler struct {
	hub     *registry.Registry
	manager *services.ManagerService
}

func NewWSHandler(hub *registry.Registry, manager *services.ManagerService) *WSHandler {
	return &WSHandler{
		hub:     hub,
		manager: manager,
	}
}

func (s *WSHandler) Handler(w http.ResponseWriter, r *http.Request) {
	log, _ := r.Context().Value(middleware.LoggerKey).(*slog.Logger)

	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
		log.ErrorContext(r.Context(), "ws handler - unauthorised missing user_id")
		http.Error(w, "Unauthorized: User ID missing", http.StatusUnauthorized)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var upgrader = websocket.Upgrader{
		ReadBufferSize:  32,
		WriteBufferSize: 32,
		CheckOrigin: func(r *http.Request) bool {
			return true // tighten later
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.ErrorContext(r.Context(), "ws handler - upgrade - ws upgrade failed", "err", err)
		return
	}
	defer conn.Close()
	websocket := ws.NewWebSocket(ctx, conn)

	convID := r.URL.Query().Get("conv_id")
	forceNew := r.URL.Query().Get("new") == "1"
	senderID, isNew, err := s.manager.HandleConnect(ctx, userID, convID, forceNew)
	if err != nil || senderID == "" {
		log.ErrorContext(r.Context(), "ws handler - handle connect - no sender id", "err", err)
		return
	}
	resp := domain.HandshakeResponse{
		Type:          domain.TypeHandshake,
		SenderID:      senderID,
		IsNewIdentity: isNew,
	}
	_ = conn.WriteJSON(resp)
	log.InfoContext(r.Context(), "ws handler - ws connection established", "sender_id", senderID)
	// Start registry and worker
	client := ws.NewClient(ctx, websocket, senderID, convID)
	s.hub.Register(client)
	defer s.manager.HandleDisconnect(ctx, senderID, convID)
	defer s.hub.Unregister(client)
	log.InfoContext(r.Context(), "ws handler - register - client updated into registry", "sender_id", senderID)
	// Heartbeat
	go s.manager.HandleHeartbeat(ctx, senderID, convID)
	log.InfoContext(r.Context(), "ws handler - handle heartbeat - heartbeat started", "sender_id", senderID)
	// Read loop
	websocket.ReadLoop(func(data []byte) {
		go func(msgData []byte) {
			s.manager.HandleMessage(ctx, senderID, convID, msgData)
		}(data)
	})
}
