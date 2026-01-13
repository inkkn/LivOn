package services

import (
	"context"
	"encoding/json"
	"errors"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type IManagerService interface {
	// HandleConnect HandleDisconnect HandleMessage HandleHeartbeat
	// HandleConnect manages the 5-min rejoin logic and initial PG update
	// Returns the assigned sender_id and previous message history metadata
	HandleConnect(ctx context.Context, userID, convID string, forceNew bool) (string, bool, error)
	// HandleDisconnect performs the final PG last_seen_at update
	HandleDisconnect(ctx context.Context, senderID string, convID string) error
	// HandleHeartbeat coordinates the 30s Redis update and the 5-min PG sync
	HandleHeartbeat(ctx context.Context, senderID string, convID string) error
	HandleHistory(ctx context.Context, convID string) []domain.Message
}

var tracer = otel.Tracer("manager-service")

type ManagerService struct {
	convRepo  domain.ConversationRepository
	presStore contracts.PresenceStore
	session   ISessionService
	message   IMessageService
	txManager *TxManager
	log       *slog.Logger
}

func NewManagerService(
	log *slog.Logger,
	convRepo domain.ConversationRepository,
	presStore contracts.PresenceStore,
	session *SessionService,
	message *MessageService,
	txManager *TxManager,
) *ManagerService {
	return &ManagerService{
		log:       log,
		convRepo:  convRepo,
		presStore: presStore,
		session:   session,
		message:   message,
		txManager: txManager,
	}
}

func (c *ManagerService) HandleConnect(
	ctx context.Context,
	userID, convID string,
	forceNew bool,
) (string, bool, error) {
	ctx, span := tracer.Start(ctx, "ManagerService.HandleConnect", trace.WithAttributes(
		attribute.String("user_id", userID),
		attribute.String("conv_id", convID),
	))
	defer span.End()
	if userID == "" || convID == "" {
		err := errors.New("invalid heartbeat parameters")
		span.RecordError(err)
		return "", false, err
	}
	var cid uuid.UUID
	if err := uuid.Validate(convID); err != nil {
		span.RecordError(err)
		c.log.ErrorContext(ctx, "manager - handle connect - wrong conv_id", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, domain.ErrInvalidConversationID
	}
	cid = uuid.MustParse(convID)
	if participants, err := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 || err != nil {
		if err := c.txManager.WithTx(ctx, func(txCtx context.Context) error {
			_, tSpan := tracer.Start(txCtx, "DB.CreateConversation")
			defer tSpan.End()
			if _, err := c.convRepo.CreateConversation(ctx, cid); err != nil {
				tSpan.RecordError(err)
				return err
			}
			return nil
		}); err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, "transaction failed")
			c.log.ErrorContext(ctx, "manager - handle connect - ensure conversation failed", "conv_id", cid.String(), "user_id", userID, "err", err)
			return "", false, err
		}
		c.log.InfoContext(ctx, "manager - handle connect - ensure conversation success", "conv_id", convID, "user_id", userID)
	}
	// Identity resolution (PG boundary)
	session, err := c.session.StartSession(ctx, userID, convID, forceNew)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "start session failed")
		c.log.ErrorContext(ctx, "manager - handle connect - start session failed", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, err
	}
	senderID := session.SenderID.String()
	isNewIdentity := session.IsNewIdentity
	// Immediate presence signal (Postgres cold path)
	if err := c.session.SessionSync(ctx, senderID, convID); err != nil {
		span.RecordError(err)
		c.log.ErrorContext(ctx, "manager - handle connect - send heartbeat failed", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, err
	}
	span.SetStatus(codes.Ok, "connected")
	return senderID, isNewIdentity, nil
}

func (c *ManagerService) HandleHeartbeat(
	ctx context.Context,
	senderID string,
	convID string,
) error {
	if senderID == "" || convID == "" {
		return errors.New("invalid heartbeat parameters")
	}
	ticker1 := time.NewTicker(30 * time.Second)
	defer ticker1.Stop()
	ticker2 := time.NewTicker(120 * time.Second)
	defer ticker2.Stop()
	for {
		select {
		case <-ctx.Done():
			c.log.Info("manager - handle heartbeat - stopped", "conv_id", convID, "sender_id", senderID)
			return nil
		case <-ticker1.C:
			_, span := tracer.Start(ctx, "Heartbeat.UpdateOnlineStatus")
			if err := c.presStore.UpdateOnlineStatus(ctx, convID, senderID, 45*time.Second); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "redis update failed")
				c.log.ErrorContext(ctx, "manager - handle heartbeat - update online status failed", "conv_id", convID, "sender_id", senderID, "err", err)
			}
			span.End()
		case <-ticker2.C:
			_, span := tracer.Start(ctx, "Heartbeat.SessionSync")
			if err := c.session.SessionSync(ctx, senderID, convID); err != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, "session sync failed")
				c.log.ErrorContext(ctx, "manager - handle heartbeat - send heartbeat failed", "conv_id", convID, "sender_id", senderID, "err", err)
			}
			span.End()
		}

	}
}

func (c *ManagerService) HandleDisconnect(
	ctx context.Context,
	senderID, convID string,
) error {
	ctx, span := tracer.Start(ctx, "ManagerService.HandleDisconnect", trace.WithAttributes(
		attribute.String("sender_id", senderID),
		attribute.String("conv_id", convID),
	))
	defer span.End()
	if senderID == "" {
		err := errors.New("invalid disconnect parameters")
		span.RecordError(err)
		return err
	}
	// Explicit leave boundary (optional but correct)
	if err := c.session.StopSession(ctx, senderID, convID); err != nil {
		span.RecordError(err)
		c.log.ErrorContext(ctx, "manager - handle disconnect - stop session failed", "conv_id", convID, "sender_id", senderID, "err", err)
		return err
	}
	if participants, _ := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 {
		if err := c.convRepo.DeleteConversation(ctx, uuid.MustParse(convID)); err != nil {
			span.RecordError(err)
			c.log.ErrorContext(ctx, "manager - handle disconnect - delete conversation failed", "conv_id", convID, "sender_id", senderID, "err", err)
		}
		if err := c.presStore.ClearConversation(ctx, convID); err != nil {
			span.RecordError(err)
			c.log.ErrorContext(ctx, "manager - handle disconnect - clear conversation failed", "conv_id", convID, "sender_id", senderID, "err", err)
		}
	}
	return nil
}

func (c *ManagerService) HandleMessage(
	ctx context.Context,
	senderID string,
	convID string,
	raw []byte,
) error {
	ctx, span := tracer.Start(ctx, "ManagerService.HandleMessage", trace.WithAttributes(
		attribute.String("sender_id", senderID),
		attribute.String("conv_id", convID),
		attribute.Int("payload_size", len(raw)),
	))
	defer span.End()
	var err error
	var in struct {
		ClientMsgID string `json:"client_msg_id"`
		Payload     string `json:"payload"`
	}
	if err = json.Unmarshal(raw, &in); err != nil {
		span.RecordError(err)
		c.log.Error("manager - handle message - wrong format", "sender_id", senderID, "conv_id", convID)
		return err
	}
	// var payload *domain.MessagePayload
	// returns payload and publishes to redis stream store until messages are persisted.
	if _, err = c.message.AcceptMessage(ctx, senderID, convID, in.Payload, in.ClientMsgID); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "accept message failed")
		c.log.ErrorContext(ctx, "manager - handle message - accept message failed", "conv_id", convID, "sender_id", senderID, "err", err)
		return err
	}
	// Unnecessary to instantly persist and send acknowledgment.
	// if err = c.message.SaveAndBroadcast(ctx, payload); err != nil {
	// 	return err
	// }
	return nil
}

func (m *ManagerService) HandleHistory(ctx context.Context, convID string) []domain.Message {
	ctx, span := tracer.Start(ctx, "ManagerService.HandleHistory", trace.WithAttributes(
		attribute.String("conv_id", convID),
	))
	defer span.End()
	var messages []domain.Message
	if err := uuid.Validate(convID); err != nil {
		span.RecordError(err)
		m.log.Error("manager - handle history - wrong conversation id", "conv_id", convID, "err", err)
		return messages
	}
	if msgs, err := m.message.GetMessages(ctx, uuid.MustParse(convID)); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "db read failed")
		m.log.ErrorContext(ctx, "manager - handle history - get messages failed", "conv_id", convID, "err", err)
		return messages
	} else {
		span.SetAttributes(attribute.Int("message_count", len(msgs)))
		m.log.InfoContext(ctx, "manager - handle history - get messages success", "conv_id", convID, "len_messages", len(msgs))
		return msgs
	}
}
