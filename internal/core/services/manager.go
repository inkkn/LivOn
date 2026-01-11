package services

import (
	"context"
	"encoding/json"
	"errors"
	"livon/internal/core/contracts"
	"livon/internal/core/domain"
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
}

type ManagerService struct {
	convRepo  domain.ConversationRepository
	presStore contracts.PresenceStore
	session   ISessionService
	message   IMessageService
}

func NewManagerService(
	convRepo domain.ConversationRepository,
	presStore contracts.PresenceStore,
	session *SessionService,
	message *MessageService,
) *ManagerService {
	return &ManagerService{
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
	cid := uuid.MustParse(convID)
	if participants, _ := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 {
		if _, err := c.convRepo.CreateConversation(ctx, cid); err != nil {
			return "", false, err
		}
	}
	// Identity resolution (PG boundary)
	session, err := c.session.StartSession(ctx, userID, convID, forceNew)
	if err != nil {
		return "", false, err
	}
	senderID := session.SenderID.String()
	isNewIdentity := session.IsNewIdentity
	// Immediate presence signal (Redis hot path)
	if err := c.session.SendHeartbeat(ctx, senderID, convID); err != nil {
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
		_ = c.presStore.UpdateOnlineStatus(ctx, convID, senderID, 45*time.Second)
		// Cold path
		_ = c.session.SendHeartbeat(ctx, senderID, convID)
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
		return err
	}
	if participants, _ := c.presStore.GetOnlineParticipants(ctx, convID); len(participants) == 0 {
		_ = c.convRepo.DeleteConversation(ctx, uuid.MustParse(convID))
		_ = c.presStore.ClearConversation(ctx, convID)
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
		return err
	}
	// var payload *domain.MessagePayload
	// returns payload and publishes to redis stream store until messages are persisted.
	if _, err = c.message.AcceptMessage(ctx, senderID, convID, in.Payload, in.ClientMsgID); err != nil {
		return err
	}
	// Unnecessary to instantly persist and send acknowledgment.
	// if err = c.message.SaveAndBroadcast(ctx, payload); err != nil {
	// 	return err
	// }
	return nil
}
