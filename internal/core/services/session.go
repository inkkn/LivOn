package services

import (
	"context"
	"livon/internal/core/domain"
	"log/slog"
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
	SessionSync(ctx context.Context, senderID string, convID string) error
}

type SessionService struct {
	memRepo   domain.ConversationParticipantRepository
	txManager *TxManager
	log       *slog.Logger
}

func NewSessionService(
	log *slog.Logger,
	memRepo domain.ConversationParticipantRepository,
	txManager *TxManager,
) *SessionService {
	return &SessionService{
		log:       log,
		memRepo:   memRepo,
		txManager: txManager,
	}
}

func (s *SessionService) StartSession(
	ctx context.Context,
	userID string,
	convID string,
	forceNew bool,
) (*domain.Session, error) {
	cid := uuid.MustParse(convID)
	var session *domain.Session
	err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		if !forceNew {
			p, err := s.memRepo.FindRecentParticipant(txCtx, userID, cid)
			if err == nil && p != nil {
				if time.Since(p.LastSeenAt) <= 3*time.Minute && p.LeftAt == nil {
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
		now := time.Now()
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
		s.log.ErrorContext(ctx, "session - start session - create participant failed", "conv_id", convID, "user_id", userID, "error", err)
		return nil, err
	}
	s.log.InfoContext(ctx, "session - start session - create participant success", "conv_id", convID, "user_id", userID, "sender_id", session.SenderID)
	return session, nil
}

func (s *SessionService) StopSession(ctx context.Context, senderID, convID string) error {
	if err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return s.memRepo.MarkLeft(txCtx, uuid.MustParse(senderID))
	}); err != nil {
		s.log.ErrorContext(ctx, "session - stop session - mark left failed", "conv_id", convID, "sender_id", senderID, "error", err)
		return err
	}
	s.log.InfoContext(ctx, "session - stop session - mark left success", "conv_id", convID, "sender_id", senderID)
	return nil
}

func (s *SessionService) SessionSync(
	ctx context.Context,
	senderID string,
	convID string,
) error {
	if err := s.txManager.WithTx(ctx, func(txCtx context.Context) error {
		return s.memRepo.UpdatePresence(txCtx, uuid.MustParse(senderID))
	}); err != nil {
		s.log.ErrorContext(ctx, "session - session sync - postgres update presence failed", "conv_id", convID, "sender_id", senderID, "err", err)
		return err
	}
	s.log.InfoContext(ctx, "session - session sync - postgres update presence success", "conv_id", convID, "sender_id", senderID)
	return nil
}
