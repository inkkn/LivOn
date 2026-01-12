package worker

import (
	"context"
	"encoding/json"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
	"livon/internal/core/services"
	"livon/internal/plugins/redis"
	"log/slog"
)

type ConversationWorker struct {
	log      *slog.Logger
	queue    redis.RedisMessageQueue
	messages *services.MessageService
	conGroup string
}

func NewConversationWorker(
	log *slog.Logger,
	queue redis.RedisMessageQueue,
	messages *services.MessageService,
	conGroup string,
) contracts.AsyncWorker {
	return &ConversationWorker{
		log:      log,
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
	w.queue.SubscribeToStream(ctx, convID, w.conGroup, w.ProcessMessage)
	w.log.InfoContext(ctx, "worker - process message - subscribe to stream success", "topic", convID, "group", w.conGroup)
	return nil
}

func (w *ConversationWorker) ProcessMessage(
	ctx context.Context,
	messageID string, // Added messageID parameter
	raw []byte,
) error {
	var payload domain.MessagePayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		w.log.Error("worker - process message - wrong payload")
		return err
	}
	if err := w.messages.SaveAndBroadcast(ctx, &payload); err != nil {
		w.log.ErrorContext(ctx, "worker - process message - save and broadcast failed", "message_id", messageID)
		return err
	}
	w.log.InfoContext(ctx, "worker - process message - save and broadcast sucess", "message_id", messageID)
	// Acknowledge the message (XACK)
	// Now that the DB save is confirmed, tell Redis we are done.
	// remove it from the Pending Entries List (PEL)
	convIDStr := payload.ConversationID.String()
	if err := w.queue.AcknowledgeMessage(ctx, convIDStr, w.conGroup, messageID); err != nil {
		w.log.ErrorContext(ctx, "worker - process message - acknowledge message failed", "message_id", messageID)
		return err
	}
	// Delete the message from the stream (XDEL)
	// This keeps the stream memory-efficient.
	if err := w.queue.DeleteMessage(ctx, convIDStr, messageID); err != nil {
		// the message is already processed and ACKed.
		w.log.ErrorContext(ctx, "worker - process message - delete message failed", "message_id", messageID)
	}
	w.log.InfoContext(ctx, "worker - process message - acknowledge and delete message sucess", "message_id", messageID)
	return nil
}
