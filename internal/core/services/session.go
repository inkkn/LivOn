package services

import (
	"context"
	"livon/internal/core/domain"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ISessionService interface {
	// StartSession determines if a user gets their old sender_id back
	// or a brand new one based on the 5-minute window or opt-out flag.
	StartSession(ctx context.Context, userID string, convID string, forceNew bool) (*domain.Session, error)
	// StopSession marks a participant as having left, breaking the 5-min link.
	StopSession(ctx context.Context, senderID, convID string) error
	// SendHeartbeat updates Redis every 30s and decides when
	// to flush 'last_seen_at' to Postgres (every 5 mins).
	SendHeartbeat(ctx context.Context, senderID string, convID string) error
}

type SessionService struct {
	memRepo   domain.ConversationParticipantRepository
	txManager *TxManager
	lastFlush sync.Map // senderID → time.Time
	clock     func() time.Time
}

func NewSessionService(
	memRepo domain.ConversationParticipantRepository,
	txManager *TxManager,
) *SessionService {
	return &SessionService{
		memRepo:   memRepo,
		txManager: txManager,
		lastFlush: sync.Map{},
		clock:     time.Now,
	}
}

func (s *SessionService) StartSession(
	ctx context.Context,
	userID string,
	convID string,
	forceNew bool,
) (*domain.Session, error) {
	cid := uuid.MustParse(convID)
	now := s.clock()
	var session *domain.Session
	err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if !forceNew {
			p, err := s.memRepo.FindRecentParticipant(txCtx, userID, cid)
			if err == nil && p != nil {
				if now.Sub(p.LastSeenAt) <= 5*time.Minute && p.LeftAt == nil {
					session = &domain.Session{
						UserID:         userID,
						ConversationID: cid,
						SenderID:       p.ID,
						JoinedAt:       p.JoinedAt,
						IsNewIdentity:  false,
					}
					return nil // transaction commits
				}
			}
		}
		// New identity logic
		p := &domain.Participant{
			ID:             uuid.New(),
			UserID:         userID,
			ConversationID: cid,
			JoinedAt:       now,
			LastSeenAt:     now,
		}
		if err := s.memRepo.CreateParticipant(txCtx, p); err != nil {
			return err
		}
		session = &domain.Session{
			UserID:         userID,
			ConversationID: cid,
			SenderID:       p.ID,
			JoinedAt:       p.JoinedAt,
			IsNewIdentity:  true,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return session, nil
}

func (s *SessionService) StopSession(ctx context.Context, senderID, convID string) error {
	return s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return s.memRepo.MarkLeft(txCtx, uuid.MustParse(senderID))
	})
}

func (s *SessionService) SendHeartbeat(
	ctx context.Context,
	senderID string,
	convID string,
) error {
	now := s.clock()
	val, _ := s.lastFlush.LoadOrStore(senderID, now)
	if now.Sub(val.(time.Time)) >= 5*time.Minute {
		if err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
			return s.memRepo.UpdatePresence(txCtx, uuid.MustParse(senderID))
		}); err != nil {
			return err
		}
		s.lastFlush.Store(senderID, now)
	}
	return nil
}
