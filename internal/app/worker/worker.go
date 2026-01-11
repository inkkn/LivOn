package worker

import (
	"context"
	"encoding/json"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
	"livon/internal/core/services"
	"livon/internal/plugins/redis"
	"livon/pkg/logging"
)

type ConversationWorker struct {
	queue    redis.RedisMessageQueue
	messages *services.MessageService
	conGroup string
}

func NewConversationWorker(
	queue redis.RedisMessageQueue,
	messages *services.MessageService,
	conGroup string,
) contracts.AsyncWorker {
	return &ConversationWorker{
		queue:    queue,
		messages: messages,
		conGroup: conGroup,
	}
}

/*
	type AsyncWorker interface {
		// Run starts the consumer loop for a specific conversation partition
		Run(ctx context.Context, convID string) error
		// ProcessMessage receives messages from redis stream for conversation topic
		// Send Ack to redis stream for message
		// Executes messages.SaveAndBrodcast method
		// Delete message from stream
		ProcessMessage(ctx context.Context, msgID string, rawData []byte) error
	}
*/

func (w *ConversationWorker) Run(
	ctx context.Context,
	convID string,
) error {
	logger := logging.FromContext(ctx)
	logger.With(logging.Conversation(convID))
	w.queue.SubscribeToStream(ctx, convID, w.conGroup, w.ProcessMessage)
	return nil
}

func (w *ConversationWorker) ProcessMessage(
	ctx context.Context,
	messageID string, // Added messageID parameter
	raw []byte,
) error {
	var payload domain.MessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return err
	}
	if err := w.messages.SaveAndBroadcast(ctx, &payload); err != nil {
		return err
	}
	// Acknowledge the message (XACK)
	// Now that the DB save is confirmed, tell Redis we are done.
	// remove it from the Pending Entries List (PEL)
	convIDStr := payload.ConversationID.String()
	if err := w.queue.AcknowledgeMessage(ctx, convIDStr, w.conGroup, messageID); err != nil {
		return err
	}
	// Delete the message from the stream (XDEL)
	// This keeps the stream memory-efficient.
	if err := w.queue.DeleteMessage(ctx, convIDStr, messageID); err != nil {
		// the message is already processed and ACKed.
		// logging.FromContext(ctx).Error("failed to delete message from stream", "id", messageID)
	}
	return nil
}
