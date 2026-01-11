package contracts

import (
	"context"
	"time"
)

// For each converation, use ZSET to store presence info
type PresenceStore interface {
	// UpdateStatus sets the TTL-based keys in Redis
	UpdateOnlineStatus(ctx context.Context, convID string, senderID string, ttl time.Duration) error
	// GetOnlineParticipants returns a list of sender_ids currently active
	GetOnlineParticipants(ctx context.Context, convID string) ([]string, error)
	// Manual clean up
	ClearConversation(ctx context.Context, convID string) error
}
