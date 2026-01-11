package ws

import (
	"context"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocket struct {
	*websocket.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

func NewWebSocket(parent context.Context, conn *websocket.Conn) *WebSocket {
	ctx, cancel := context.WithCancel(parent)
	return &WebSocket{Conn: conn, ctx: ctx, cancel: cancel}
}

func (w *WebSocket) WriteMessage(data []byte) error {
	w.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return w.Conn.WriteMessage(websocket.TextMessage, data)
}

func (w *WebSocket) ReadLoop(onMsg func([]byte)) {
	// Ensure cleanup happens when the loop breaks
	defer func() {
		w.Close()
	}()

	// Configure Read Limits (Protects against memory exhaustion)
	w.Conn.SetReadLimit(512 * 1024) // 512KB max message size

	for {
		_, data, err := w.Conn.ReadMessage()
		if err != nil {
			// Check if it's a clean closure or an error
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("Unexpected close error: %v", err)
			}
			break // Exit the loop
		}

		if len(data) > 0 {
			onMsg(data)
		}
	}
}

func (w *WebSocket) Close() {
	w.cancel()
	_ = w.Conn.Close()
}
