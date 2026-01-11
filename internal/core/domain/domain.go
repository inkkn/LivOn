package domain

import (
	"time"

	"github.com/google/uuid"
)

// User represents the internal identity (e.g., tied to a phone number)
type User struct {
	ID        string
	CreatedAt time.Time
}

func NewUser(id string) *User {
	return &User{
		ID:        id,
		CreatedAt: time.Now(),
	}
}

// Conversation represents a chat room
type Conversation struct {
	ID        uuid.UUID
	CreatedAt time.Time
}

func NewConversation() (*Conversation, error) {
	var id uuid.UUID
	var err error
	if id, err = uuid.NewUUID(); err != nil {
		return &Conversation{}, err
	}
	return &Conversation{
		ID:        id,
		CreatedAt: time.Now(),
	}, nil
}

// Participant represents the "Privacy Bridge" (The ephemeral sender_id)
type Participant struct {
	ID             uuid.UUID // This is the 'sender_id' shown to others
	ConversationID uuid.UUID
	UserID         string
	JoinedAt       time.Time
	LastSeenAt     time.Time
	LeftAt         *time.Time // Nullable
}

// Message represents a chat entry with its ordering sequence
type Message struct {
	ID             uuid.UUID
	ConversationID uuid.UUID
	SenderID       uuid.UUID // Refers to Participant.ID
	Seq            int64     // The strict monotonic counter
	Payload        string
	CreatedAt      time.Time
}

// Session represents the active connection context for a user in a room.
// It bridges the gap between the permanent User and the anonymous Participant.
type Session struct {
	UserID         string
	ConversationID uuid.UUID
	SenderID       uuid.UUID // The Participant.ID (anonymous)
	JoinedAt       time.Time
	IsNewIdentity  bool // Useful for the frontend to know if identity changed
}
