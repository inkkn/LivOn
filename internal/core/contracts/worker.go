package contracts

import "context"

type AsyncWorker interface {
	// Run starts the consumer loop for a specific conversation or partition
	Run(ctx context.Context, convID string) error
	// ProcessMessage receives messages from redis stream for conversation topic
	// Send Ack to redis stream for message.
	// Executes messages.SaveAndBrodcast method
	// Delete message from stream
	ProcessMessage(ctx context.Context, msgID string, rawData []byte) error
}
