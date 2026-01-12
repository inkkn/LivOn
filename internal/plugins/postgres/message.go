package postgres

import (
	"context"
	"database/sql"
	"livon/internal/core/domain"

	"github.com/google/uuid"
)

type MessageRepo struct {
	db *sql.DB
}

func NewMessageRepo(db *sql.DB) *MessageRepo {
	return &MessageRepo{
		db: db,
	}
}

/*
	type MessageRepository interface {
		// Atomic Persistence: Increments sequence and inserts message in one TX
		// This fulfills the "Double Tick" requirement by returning the final Seq
		SaveWithSequence(ctx context.Context, msg *Message) (seq int64, err error)
		// Visibility Logic: Join-Onward + Recent 1-min Window
		// Uses the Participant.JoinedAt and current time to filter history
		GetVisibleMessages(ctx context.Context, convID uuid.UUID, p *Participant) ([]Message, error)
	}
*/

func (r *MessageRepo) SaveWithSequence(
	ctx context.Context,
	msg *domain.Message,
) (int64, error) {
	if msg.ConversationID == uuid.Nil {
		return 0, domain.ErrInvalidConversationID
	}
	exec := GetExecutor(ctx, r.db)
	var seq int64
	err := exec.QueryRowContext(ctx, `
        UPDATE conversation_sequences
        SET last_seq = last_seq + 1
        WHERE conversation_id = $1
        RETURNING last_seq
    `, msg.ConversationID).Scan(&seq)
	if err != nil {
		if err == sql.ErrNoRows {
			// No sequence row = conversation does not exist or not initialized
			return 0, domain.ErrSequenceNotInitialized
		}
		return 0, err
	}
	_, err = exec.ExecContext(ctx, `
        INSERT INTO messages (
            id, conversation_id, sender_id, seq, payload
        ) VALUES ($1, $2, $3, $4, $5)
    `,
		msg.ID,
		msg.ConversationID,
		msg.SenderID,
		seq,
		msg.Payload,
	)
	if err != nil {
		return 0, err
	}
	return seq, nil
}

func (r *MessageRepo) GetVisibleMessages(
	ctx context.Context,
	convID uuid.UUID,
) ([]domain.Message, error) {
	if convID == uuid.Nil {
		return nil, domain.ErrInvalidConversationID
	}
	exec := GetExecutor(ctx, r.db)
	rows, err := exec.QueryContext(ctx, `
		SELECT id, conversation_id, sender_id, seq, payload, created_at
		FROM messages
		WHERE conversation_id = $1
		AND created_at >= now() - interval '1 minutes'
		ORDER BY seq ASC
	`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []domain.Message
	for rows.Next() {
		var m domain.Message
		if err := rows.Scan(
			&m.ID,
			&m.ConversationID,
			&m.SenderID,
			&m.Seq,
			&m.Payload,
			&m.CreatedAt,
		); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, nil
}
