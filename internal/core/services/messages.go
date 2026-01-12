package services

import (
	"context"
	"encoding/json"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

type IMessageService interface {
	// ProcessMessage validates the message and optionally sends to redis stream
	// Sends a Domain AckMessage to trigger the UI "Single Tick"
	AcceptMessage(ctx context.Context, senderID string, convID string, payload string, clientMsgID string) (domain.MessagePayload, error)
	// SaveAndBroadcast runs the atomic DB sequence logic and optionally sends to redis pubsub
	// After DB commit, it triggers the "Double Tick"
	SaveAndBroadcast(ctx context.Context, payload *domain.MessagePayload) error
	// GetMessages calculates the visibility window: max(joined_at, now - 1min)
	// and returns filtered messages.
	GetMessages(ctx context.Context, convID uuid.UUID) ([]domain.Message, error)
}

type MessageService struct {
	queue     contracts.MessageQueue // For now no use of redis message queue in message delivery
	registry  contracts.Registry
	Repo      domain.MessageRepository
	txManager *TxManager
	log       *slog.Logger
}

func NewMessageService(
	log *slog.Logger,
	queue contracts.MessageQueue,
	registry contracts.Registry,
	repo domain.MessageRepository,
	txManager *TxManager,
) *MessageService {
	return &MessageService{
		log:       log,
		queue:     queue,
		registry:  registry,
		Repo:      repo,
		txManager: txManager,
	}
}

func (w *MessageService) AcceptMessage(
	ctx context.Context,
	senderID string,
	convID string,
	payload string,
	clientMsgID string,
) (domain.MessagePayload, error) {
	message_payload := domain.MessagePayload{
		ClientMsgID:    clientMsgID,
		ConversationID: uuid.MustParse(convID),
		SenderID:       uuid.MustParse(senderID),
		Payload:        payload,
		CreatedAt:      time.Now(),
	}
	// Single tick (only to sender)
	ack := domain.AckMessage{
		Type:        domain.TypeAck,
		ClientMsgID: clientMsgID,
		Status:      domain.AckServerReceived,
		Timestamp:   time.Now(),
	}
	raw, _ := json.Marshal(message_payload)
	if err := w.queue.PublishToStream(ctx, convID, raw); err != nil {
		w.log.ErrorContext(ctx, "messages - accept message - publish to stream failed", "stream", convID, "error", err)
		return domain.MessagePayload{}, err
	}
	w.log.InfoContext(ctx, "messages - accept message - publish to stream success", "stream", convID)
	w.registry.SendAck(ctx, senderID, ack)
	return message_payload, nil
}

// executes the atomic DB transaction:
// 1. Incr Sequence -> 2. Insert Msg -> 3. Broadcast Double Tick -> 4. Ack Stream
func (w *MessageService) SaveAndBroadcast(
	ctx context.Context,
	payload *domain.MessagePayload,
) error {
	msg := &domain.Message{
		ID:             uuid.New(),
		ConversationID: payload.ConversationID,
		SenderID:       payload.SenderID,
		Payload:        payload.Payload,
		CreatedAt:      payload.CreatedAt,
	}
	var seq int64
	if err := w.txManager.WithTx(ctx, func(txCtx context.Context) error {
		var txErr error
		seq, txErr = w.Repo.SaveWithSequence(txCtx, msg)
		return txErr
	}); err != nil {
		w.log.ErrorContext(ctx, "messages - save and broadcast - save with sequence failed", "err", err)
		return err
	}
	w.log.InfoContext(ctx, "messages - save and broadcast - save with sequence success", "seq", seq, "conv_id", msg.ConversationID, "sender_id", msg.SenderID)
	msg.Seq = seq
	// Broadcast message
	out := domain.ChatMessage{
		Type:           domain.TypeMessage,
		ConversationID: msg.ConversationID.String(),
		SenderID:       msg.SenderID.String(),
		Seq:            msg.Seq,
		Payload:        msg.Payload,
		CreatedAt:      msg.CreatedAt,
	}
	// Double tick (only to sender)
	ack := domain.AckMessage{
		Type:        domain.TypeAck,
		ClientMsgID: payload.ClientMsgID,
		Status:      domain.AckPersisted,
		Seq:         msg.Seq,
		Timestamp:   time.Now(),
	}
	w.registry.Broadcast(ctx, msg.ConversationID.String(), out)
	w.registry.SendAck(ctx, msg.SenderID.String(), ack)
	return nil
}

func (m *MessageService) GetMessages(ctx context.Context, cid uuid.UUID) ([]domain.Message, error) {
	var no_msg []domain.Message
	var msgs []domain.Message
	var err error
	if er := m.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if msgs, err = m.Repo.GetVisibleMessages(ctx, cid); err != nil {
			return err
		} else {
			return nil
		}
	}); er != nil {
		m.log.ErrorContext(ctx, "messages - get messages - get visible messages failed", "conv_id", cid.String())
		return no_msg, er
	} else {
		m.log.InfoContext(ctx, "messages - get messages - get visible messages sucesss", "conv_id", cid.String())
		return msgs, nil
	}
}
