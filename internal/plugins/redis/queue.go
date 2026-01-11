package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type RedisMessageQueue struct {
	rdb *redis.Client
}

func NewRedisMessageQueue(rdb *redis.Client) *RedisMessageQueue {
	return &RedisMessageQueue{rdb: rdb}
}

/*
	type MessageQueue interface {
		// Producer side (Ingest Service)
		PublishToStream(ctx context.Context, topic, payload []byte) error
		// Consumer side (Worker Service)
		// SubscribeToStream handles the reliable reading from the Redis Stream
		SubscribeToStream(ctx context.Context, topic string, conGroup string, handler func(ctx context.Context, messageID string, data []byte) error) error
		// AcknowledgeMessage acknowledges redis stream that message is picked for processing
		AcknowledgeMessage(ctx context.Context, convID, conGroup, mesgID string) error
		// DeleteStream removes the message stream from redis
		DeleteStream(ctx context.Context, convID string) error
		// Deletes Message from redis stream
		DeleteMessage(ctx context.Context, convID, mesgID string) error
	}
*/

func (q *RedisMessageQueue) streamKey(convID string) string {
	return "stream:" + convID
}

func (q *RedisMessageQueue) PublishToStream(ctx context.Context, convID string, payload []byte) error {
	return q.rdb.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey(convID),
		MaxLen: 1000,
		Approx: true,
		ID:     "*",
		Values: map[string]interface{}{"data": payload},
	}).Err()
}

func (q *RedisMessageQueue) SubscribeToStream(
	ctx context.Context,
	convID string,
	conGroup string,
	handler func(ctx context.Context, messageID string, data []byte) error,
) error {
	topic := q.streamKey(convID)
	// Create group if not exists
	err := q.rdb.XGroupCreateMkStream(ctx, topic, conGroup, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return fmt.Errorf("failed to create consumer group: %w", err)
	}
	consumerName := uuid.NewString()
	// Run in a goroutine
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Read new messages (">")
				res, err := q.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    conGroup,
					Consumer: consumerName,
					Streams:  []string{topic, ">"},
					Count:    1,
					Block:    2 * time.Second,
				}).Result()
				if err != nil {
					if err != redis.Nil {
						log.Printf("Stream read error: %v", err)
					}
					continue
				}
				for _, stream := range res {
					for _, msg := range stream.Messages {
						raw, ok := msg.Values["data"].(string)
						if !ok {
							continue
						}
						if err := handler(ctx, msg.ID, []byte(raw)); err != nil {
							log.Printf("Handler error for message %s: %v", msg.ID, err)
						}
					}
				}
			}
		}
	}()
	return nil
}

func (q *RedisMessageQueue) AcknowledgeMessage(ctx context.Context, convID, conGroup, mesgID string) error {
	return q.rdb.XAck(ctx, q.streamKey(convID), conGroup, mesgID).Err()
}

func (q *RedisMessageQueue) DeleteMessage(ctx context.Context, convID, mesgID string) error {
	return q.rdb.XDel(ctx, q.streamKey(convID), mesgID).Err()
}

func (q *RedisMessageQueue) DeleteStream(ctx context.Context, convID string) error {
	return q.rdb.Del(ctx, q.streamKey(convID)).Err()
}

/*
// Broadcast/Unicast (Pub/Sub)

func (q *RedisMessageQueue) ToConversation(ctx context.Context, convID string, msg any) error {
	raw, _ := json.Marshal(msg)
	return q.rdb.Publish(ctx, "room:"+convID, raw).Err()
}

func (q *RedisMessageQueue) ToSender(ctx context.Context, senderID string, convID string, msg any) error {
	raw, _ := json.Marshal(msg)
	return q.rdb.Publish(ctx, "sender:"+senderID, raw).Err()
}

// ListenToConversation subscribes to a conversation channel and returns a channel of messages.
func (q *RedisMessageQueue) ListenToConversation(ctx context.Context, convID string) (<-chan *redis.Message, error) {
	pubsub := q.rdb.Subscribe(ctx, "room:"+convID)

	// return the Channel() which handles the background receiving
	return pubsub.Channel(), nil
}

// ListenToSender subscribes to a user-specific channel and returns a channel of messages.
func (q *RedisMessageQueue) ListenToSender(ctx context.Context, senderID string) (<-chan *redis.Message, error) {
	pubsub := q.rdb.Subscribe(ctx, "sender:"+senderID)

	return pubsub.Channel(), nil
}
*/
