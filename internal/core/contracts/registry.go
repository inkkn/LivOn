package contracts

import (
	"context"
	"livon/internal/core/domain"
)

// Registry defines the global orchestration layer that manages physical
// client connections and bridges Redis events to the local hubs.
type Registry interface {
	// Register adds a client to the local node memory and joins them to their room.
	Register(c Client)
	// Unregister removes the client and cleans up their room participation.
	Unregister(c Client)
	// SendAck targets a specific local client to deliver a received or delivery confirmation.
	SendAck(ctx context.Context, senderID string, ack domain.AckMessage)
	// Broadcast sends a message to all local clients in a room except the sender.
	Broadcast(ctx context.Context, convID string, msg domain.ChatMessage)
}

// Client represents the minimal interface required for the Registry to
// communicate with an individual WebSocket connection.
type Client interface {
	SenderID() string
	ConversationID() string
	Send(ctx context.Context, data []byte) error
	Close()
}
