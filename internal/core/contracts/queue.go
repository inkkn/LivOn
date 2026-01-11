package contracts

import (
	"context"
)

type MessageQueue interface {
	// Producer side (Ingest Service)
	PublishToStream(ctx context.Context, topic string, payload []byte) error
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
