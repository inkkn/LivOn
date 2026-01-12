package postgres

import (
	"context"
	"database/sql"
	"livon/internal/core/domain"

	"github.com/google/uuid"
)

type ConversationRepo struct {
	db *sql.DB
}

func NewConversationRepo(db *sql.DB) *ConversationRepo {
	return &ConversationRepo{db: db}
}

/*
	-- Conversations
	CREATE TABLE conversations (
		id          UUID PRIMARY KEY,
		created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	type ConversationRepository interface {
		GetConversationByID(ctx context.Context, convID uuid.UUID) (*Conversation, error)
		CreateConversation(ctx context.Context, convID uuid.UUID) (*Conversation, error)
		DeleteConversation(ctx context.Context, convID uuid.UUID) error
	}
*/

func (r *ConversationRepo) GetConversationByID(ctx context.Context, convID uuid.UUID) (*domain.Conversation, error) {
	if convID == uuid.Nil {
		return nil, domain.ErrInvalidConversationID
	}
	conversation := &domain.Conversation{ID: convID}
	query := `SELECT created_at FROM conversations WHERE id = $1`
	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, convID).Scan(&conversation.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrConversationNotFound
		}
		return nil, err
	}
	return conversation, nil
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, convID uuid.UUID) (*domain.Conversation, error) {
	if convID == uuid.Nil {
		return nil, domain.ErrInvalidConversationID
	}
	conversation := &domain.Conversation{
		ID: convID,
	}
	// Insert new conversation or do nothing if conversation.ID already exists
	// We return the created_at to populate our core model
	query :=
		`INSERT INTO conversations (id) 
        VALUES ($1) 
		ON CONFLICT (id) DO NOTHING
        RETURNING created_at`

	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, convID).Scan(&conversation.CreatedAt)
	if err == sql.ErrNoRows {
		existing, err := r.GetConversationByID(ctx, convID)
		if err != nil {
			return nil, err
		}
		conversation.CreatedAt = existing.CreatedAt
	} else if err != nil {
		return nil, err
	}
	_, err = exec.ExecContext(ctx, `
		INSERT INTO conversation_sequences (conversation_id, last_seq)
		VALUES ($1, 0)
		ON CONFLICT (conversation_id) DO NOTHING
	`, convID)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

func (r *ConversationRepo) DeleteConversation(ctx context.Context, convID uuid.UUID) error {
	if convID == uuid.Nil {
		return domain.ErrInvalidConversationID
	}
	query := `DELETE FROM conversations WHERE id = $1`
	exec := GetExecutor(ctx, r.db)
	result, err := exec.ExecContext(ctx, query, convID)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrConversationNotFound
	}
	return nil
}
