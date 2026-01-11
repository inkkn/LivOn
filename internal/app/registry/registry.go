package registry

import (
	"context"
	"encoding/json"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
	"sync"
)

type Registry struct {
	mu         sync.RWMutex
	clients    map[string]contracts.Client // sender_id → client
	room_hub   map[string]map[string]contracts.Client
	workers    map[string]context.CancelFunc
	run_worker func(ctx context.Context, convID string) error
}

func NewRegistry() *Registry {
	return &Registry{
		clients:  make(map[string]contracts.Client),
		room_hub: make(map[string]map[string]contracts.Client),
		workers:  make(map[string]context.CancelFunc),
	}
}

func (h *Registry) RunWorker(run_worker func(ctx context.Context, convID string) error) {
	h.run_worker = run_worker
}

func (h *Registry) Register(c contracts.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	convID := c.ConversationID()
	senderID := c.SenderID()
	if h.room_hub[convID] == nil {
		h.room_hub[convID] = make(map[string]contracts.Client)
		ctx, cancel := context.WithCancel(context.Background())
		h.workers[convID] = cancel
		go h.run_worker(ctx, convID)
	}
	h.room_hub[convID][senderID] = c
	h.clients[senderID] = c
}

func (h *Registry) Unregister(c contracts.Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	convID := c.ConversationID()
	senderID := c.SenderID()
	delete(h.room_hub[convID], senderID)
	delete(h.clients, senderID)
	if len(h.room_hub[convID]) == 0 {
		delete(h.room_hub, convID)
		// stop worker
		if cancel := h.workers[convID]; cancel != nil {
			cancel()
			delete(h.workers, convID)
		}
	}
}

func (h *Registry) SendAck(ctx context.Context, senderID string, ack domain.AckMessage) {
	h.mu.RLock()
	c := h.clients[senderID]
	h.mu.RUnlock()
	if c == nil {
		return
	}
	data, _ := json.Marshal(ack)
	_ = c.Send(ctx, data)
}

func (h *Registry) Broadcast(ctx context.Context, convID string, msg domain.ChatMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	data, _ := json.Marshal(msg)
	for sid, c := range h.room_hub[convID] {
		if sid == msg.SenderID {
			continue
		}
		_ = c.Send(ctx, data)
	}
}
