package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisPresenceStore struct {
	rdb *redis.Client
}

func NewRedisPresenceStore(rdb *redis.Client) *RedisPresenceStore {
	return &RedisPresenceStore{
		rdb: rdb,
	}
}

/*
	type PresenceStore interface {
		// UpdateStatus sets the TTL-based keys in Redis
		UpdateOnlineStatus(ctx context.Context, convID string, senderID string, ttl time.Duration) error
		// GetOnlineParticipants returns a list of sender_ids currently active
		GetOnlineParticipants(ctx context.Context, convID string) ([]string, error)
		// Manual clean up
		ClearConversation(ctx context.Context, convID string) error
	}
*/

// UpdateOnlineStatus adds/updates a user in the conversation's ZSet with the current timestamp.
func (p *RedisPresenceStore) UpdateOnlineStatus(
	ctx context.Context,
	convID string,
	senderID string,
	ttl time.Duration, // "inactivity threshold"
) error {
	key := "presence:" + convID
	now := time.Now().Unix()

	// Add/Update user with current timestamp
	err := p.rdb.ZAdd(ctx, key, redis.Z{
		Score:  float64(now),
		Member: senderID,
	}).Err()
	if err != nil {
		return err
	}

	// Set an expiration on the whole ZSet so it doesn't leak memory
	// if the conversation becomes inactive.
	return p.rdb.Expire(ctx, key, ttl*2).Err()
}

// GetOnlineParticipants returns users who have checked in within the last 'ttl' duration.
func (p *RedisPresenceStore) GetOnlineParticipants(
	ctx context.Context,
	convID string,
) ([]string, error) {
	key := "presence:" + convID

	// We'll define "online" as anyone who updated in the last 2 minutes (or your preferred duration)
	// For now assume a 30-second window.
	threshold := time.Now().Add(-30 * time.Second).Unix()

	// Remove stale members first (Self-cleaning)
	p.rdb.ZRemRangeByScore(ctx, key, "-inf", strconv.FormatInt(threshold, 10))

	// Get all members remaining in the set
	return p.rdb.ZRange(ctx, key, 0, -1).Result()
}

// ClearConversation deletes the entire ZSet for the conversation.
func (p *RedisPresenceStore) ClearConversation(ctx context.Context, convID string) error {
	key := "presence:" + convID
	return p.rdb.Del(ctx, key).Err()
}
