package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository handles the persistent identity
type UserRepository interface {
	GetUserByID(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, id string) (*User, error)
	DeleteUser(ctx context.Context, id string) error
}

// Conversation repository handles conversation lifecycle.
type ConversationRepository interface {
	GetConversationByID(ctx context.Context, convID uuid.UUID) (*Conversation, error)
	CreateConversation(ctx context.Context, convID uuid.UUID) (*Conversation, error)
	DeleteConversation(ctx context.Context, convID uuid.UUID) error
}

// ConversationParticipantRepository handles the Privacy Bridge and Presence
type ConversationParticipantRepository interface {
	// Rejoin Logic - finds active session within the 5-min window
	FindRecentParticipant(ctx context.Context, userID string, convID uuid.UUID) (*Participant, error)
	// Identity Creation - Assigns a new sender_id (Participant.ID)
	CreateParticipant(ctx context.Context, p *Participant) error
	// Presence - High-durability last_seen_at sync (the 5-min PG sync)
	UpdatePresence(ctx context.Context, participantID uuid.UUID) error
	// Mark permanent leave left_at
	MarkLeft(ctx context.Context, participantID uuid.UUID) error
}

// MessageRepository handles Persistence and Guaranteed Ordering
type MessageRepository interface {
	// Atomic Persistence: Increments sequence and inserts message in one TX
	// This fulfills the "Double Tick" requirement by returning the final Seq
	SaveWithSequence(ctx context.Context, msg *Message) (seq int64, err error)
	// Visibility Logic: Join-Onward + Recent 1-min Window
	// Uses the Participant.JoinedAt and current time to filter history
	GetVisibleMessages(ctx context.Context, convID uuid.UUID) ([]Message, error)
}
