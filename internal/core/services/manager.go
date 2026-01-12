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

type ManagerService struct {
	convRepo  domain.ConversationRepository
	presStore contracts.PresenceStore
	session   ISessionService
	message   IMessageService
	log       *slog.Logger
}

func NewManagerService(
	log *slog.Logger,
	convRepo domain.ConversationRepository,
	presStore contracts.PresenceStore,
	session *SessionService,
	message *MessageService,
) *ManagerService {
	return &ManagerService{
		log:       log,
		convRepo:  convRepo,
		presStore: presStore,
		session:   session,
		message:   message,
	}
}

func (c *ManagerService) HandleConnect(
	ctx context.Context,
	userID, convID string,
	forceNew bool,
) (string, bool, error) {
	if userID == "" || convID == "" {
		return "", false, errors.New("invalid heartbeat parameters")
	}
	var cid uuid.UUID
	if err := uuid.Validate(convID); err != nil {
		c.log.ErrorContext(ctx, "manager - handle connect - wrong conv_id", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, domain.ErrInvalidConversationID
	}
	cid = uuid.MustParse(convID)
	if participants, err := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 || err != nil {
		if conv, err := c.convRepo.CreateConversation(ctx, cid); err != nil {
			c.log.ErrorContext(ctx, "manager - handle connect - create conversation failed", "conv_id", cid.String(), "user_id", userID, "err", err)
			return "", false, err
		} else {
			convID = conv.ID.String()
			cid = conv.ID
			c.log.InfoContext(ctx, "manager - handle connect - create conversation success", "conv_id", convID, "user_id", userID)
		}
	}
	// Identity resolution (PG boundary)
	session, err := c.session.StartSession(ctx, userID, convID, forceNew)
	if err != nil {
		c.log.ErrorContext(ctx, "manager - handle connect - start session failed", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, err
	}
	senderID := session.SenderID.String()
	isNewIdentity := session.IsNewIdentity
	// Immediate presence signal (Redis hot path)
	if err := c.session.SendHeartbeat(ctx, senderID, convID); err != nil {
		c.log.ErrorContext(ctx, "manager - handle connect - send heartbeat failed", "conv_id", convID, "user_id", userID, "err", err)
		return "", false, err
	}
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
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		// Hot path
		if err := c.presStore.UpdateOnlineStatus(ctx, convID, senderID, 45*time.Second); err != nil {
			c.log.ErrorContext(ctx, "manager - handle heartbeat - update online status failed", "conv_id", convID, "sender_id", senderID, "err", err)
		}
		// Cold path
		if err := c.session.SendHeartbeat(ctx, senderID, convID); err != nil {
			c.log.ErrorContext(ctx, "manager - handle heartbeat - send heartbeat failed", "conv_id", convID, "sender_id", senderID, "err", err)
		}
	}
	return nil
}

func (c *ManagerService) HandleDisconnect(
	ctx context.Context,
	senderID, convID string,
) error {
	if senderID == "" {
		return errors.New("invalid disconnect parameters")
	}
	// Explicit leave boundary (optional but correct)
	if err := c.session.StopSession(ctx, senderID, convID); err != nil {
		c.log.ErrorContext(ctx, "manager - handle disconnect - stop session failed", "conv_id", convID, "sender_id", senderID, "err", err)
		return err
	}
	if participants, _ := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 {
		if err := c.convRepo.DeleteConversation(ctx, uuid.MustParse(convID)); err != nil {
			c.log.ErrorContext(ctx, "manager - handle disconnect - delete conversation failed", "conv_id", convID, "sender_id", senderID, "err", err)
		}
		if err := c.presStore.ClearConversation(ctx, convID); err != nil {
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
	var err error
	var in struct {
		ClientMsgID string `json:"client_msg_id"`
		Payload     string `json:"payload"`
	}
	if err = json.Unmarshal(raw, &in); err != nil {
		c.log.Error("manager - handle message - wrong format", "sender_id", senderID, "conv_id", convID)
		return err
	}
	// var payload *domain.MessagePayload
	// returns payload and publishes to redis stream store until messages are persisted.
	if _, err = c.message.AcceptMessage(ctx, senderID, convID, in.Payload, in.ClientMsgID); err != nil {
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
	var messages []domain.Message
	if err := uuid.Validate(convID); err != nil {
		m.log.Error("manager - handle history - wrong conversation id", "conv_id", convID, "err", err)
		return messages
	}
	if msgs, err := m.message.GetMessages(ctx, uuid.MustParse(convID)); err != nil {
		m.log.ErrorContext(ctx, "manager - handle history - get messages failed", "conv_id", convID, "err", err)
		return messages
	} else {
		m.log.InfoContext(ctx, "manager - handle history - get messages success", "conv_id", convID, "len_messages", len(msgs))
		return msgs
	}
}
