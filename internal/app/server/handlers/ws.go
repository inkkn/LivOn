package handlers

import (
	"context"
	"livon/internal/app/registry"
	"livon/internal/app/server/ws"
	"livon/internal/core/domain"
	"livon/internal/core/services"
	"livon/pkg/middleware"
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
	userID, ok := r.Context().Value(middleware.UserIDKey).(string)
	if !ok {
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
		return
	}
	defer conn.Close()
	websocket := ws.NewWebSocket(ctx, conn)

	convID := r.URL.Query().Get("conv_id")
	forceNew := r.URL.Query().Get("new") == "1"
	senderID, isNew, err := s.manager.HandleConnect(ctx, userID, convID, forceNew)
	if err != nil || senderID == "" {
		return
	}
	resp := domain.HandshakeResponse{
		Type:          domain.TypeHandshake,
		SenderID:      senderID,
		IsNewIdentity: isNew,
	}
	_ = conn.WriteJSON(resp)

	// Start registry and worker
	client := ws.NewClient(ctx, websocket, senderID, convID)
	s.hub.Register(client)
	defer s.manager.HandleDisconnect(ctx, senderID, convID)
	defer s.hub.Unregister(client)

	// Heartbeat
	go s.manager.HandleHeartbeat(ctx, senderID, convID)

	// Read loop
	websocket.ReadLoop(func(data []byte) {
		go func(msgData []byte) {
			s.manager.HandleMessage(ctx, senderID, convID, msgData)
		}(data)
	})
}
