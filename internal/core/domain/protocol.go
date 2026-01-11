package domain

import (
	"time"

	"github.com/google/uuid"
)

const (
	TypeAck       = "ack"
	TypeMessage   = "message"
	TypePresence  = "presence"
	TypeHandshake = "handshake"
	TypeError     = "error"
)

type AckStatus string

const (
	AckServerReceived AckStatus = "server_received"
	AckPersisted      AckStatus = "persisted"
)

// HandshakeResponse is sent once on connect
type HandshakeResponse struct {
	Type          string `json:"type"` // "handshake"
	SenderID      string `json:"sender_id"`
	IsNewIdentity bool   `json:"is_new_identity"`
}

// MessagePayload structure received after processing user message.
type MessagePayload struct {
	ClientMsgID    string    `json:"client_msg_id"`
	ConversationID uuid.UUID `json:"conversation_id"`
	SenderID       uuid.UUID `json:"sender_id"`
	Payload        string    `json:"payload"`
	CreatedAt      time.Time `json:"created_at"`
}

// AckMessage is sent ONLY to the sender
type AckMessage struct {
	Type        string    `json:"type"` // always "ack"
	ClientMsgID string    `json:"client_msg_id"`
	Status      AckStatus `json:"status"`
	Seq         int64     `json:"seq,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// ChatMessage is broadcast to room subscribers
type ChatMessage struct {
	Type           string    `json:"type"` // "message"
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Seq            int64     `json:"seq"`
	Payload        string    `json:"payload"`
	CreatedAt      time.Time `json:"created_at"`
}

// PresenceEvent is pushed to room
type PresenceEvent struct {
	Type   string   `json:"type"` // "presence"
	Online []string `json:"online_sender_ids"`
}

// ErrorMessage is WS-safe error
type ErrorMessage struct {
	Type    string `json:"type"` // "error"
	Code    string `json:"code"`
	Message string `json:"message"`
}
