package postgres

import (
	"context"
	"database/sql"
	"livon/internal/core/domain"

	"github.com/google/uuid"
)

type ParticipantRepo struct {
	db *sql.DB
}

func NewParticipantRepo(db *sql.DB) *ParticipantRepo {
	return &ParticipantRepo{db: db}
}

/*
	type ConversationParticipantRepository interface {
		// Rejoin Logic - finds active session within the 5-min window
		FindRecentParticipant(ctx context.Context, userID string, convID uuid.UUID) (*Participant, error)
		// Identity Creation - Assigns a new sender_id (Participant.ID)
		CreateParticipant(ctx context.Context, p *Participant) error
		// Presence - High-durability last_seen_at sync (the 5-min PG sync)
		UpdatePresence(ctx context.Context, participantID uuid.UUID) error
		// Mark permanent leave left_at
		MarkLeft(ctx context.Context, participantID uuid.UUID) error
	}
*/

func (r *ParticipantRepo) FindRecentParticipant(
	ctx context.Context,
	userID string,
	convID uuid.UUID,
) (*domain.Participant, error) {
	if convID == uuid.Nil {
		return nil, domain.ErrInvalidConversationID
	}
	exec := GetExecutor(ctx, r.db)
	row := exec.QueryRowContext(ctx, `
		SELECT id, conversation_id, user_id, joined_at, last_seen_at, left_at
		FROM conversation_participants
		WHERE user_id = $1
		  AND conversation_id = $2
		  AND left_at IS NULL
		ORDER BY last_seen_at DESC
		LIMIT 1
	`, userID, convID)
	var p domain.Participant
	err := row.Scan(
		&p.ID,
		&p.ConversationID,
		&p.UserID,
		&p.JoinedAt,
		&p.LastSeenAt,
		&p.LeftAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &p, err
}

func (r *ParticipantRepo) CreateParticipant(
	ctx context.Context,
	p *domain.Participant,
) error {
	if p.ID == uuid.Nil {
		return domain.ErrInvalidParticipantID
	}
	if p.ConversationID == uuid.Nil {
		return domain.ErrInvalidConversationID
	}
	exec := GetExecutor(ctx, r.db)
	_, err := exec.ExecContext(ctx, `
		INSERT INTO conversation_participants (
			id, conversation_id, user_id, joined_at, last_seen_at
		) VALUES ($1, $2, $3, $4, $5)
	`,
		p.ID,
		p.ConversationID,
		p.UserID,
		p.JoinedAt,
		p.LastSeenAt,
	)
	return err
}

func (r *ParticipantRepo) UpdatePresence(
	ctx context.Context,
	participantID uuid.UUID,
) error {
	if participantID == uuid.Nil {
		return domain.ErrInvalidParticipantID
	}
	exec := GetExecutor(ctx, r.db)
	result, err := exec.ExecContext(ctx, `
		UPDATE conversation_participants
		SET last_seen_at = now()
		WHERE id = $1
	`, participantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrParticipantNotFound
	}
	return err
}

func (r *ParticipantRepo) MarkLeft(
	ctx context.Context,
	participantID uuid.UUID,
) error {
	if participantID == uuid.Nil {
		return domain.ErrInvalidParticipantID
	}
	exec := GetExecutor(ctx, r.db)
	result, err := exec.ExecContext(ctx, `
		UPDATE conversation_participants
		SET left_at = now(), last_seen_at = now()
		WHERE id = $1 AND left_at IS NULL
	`, participantID)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrParticipantNotFound
	}
	return err
}
