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
	conversation := &domain.Conversation{ID: convID}
	query := `SELECT created_at FROM conversations WHERE id = $1`
	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, convID).Scan(&conversation.CreatedAt)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

func (r *ConversationRepo) CreateConversation(ctx context.Context, convID uuid.UUID) (*domain.Conversation, error) {
	conversation := &domain.Conversation{
		ID: convID,
	}
	// Insert new conversation or do nothing if conversation.ID already exists
	// We return the created_at to populate our core model
	query :=
		`INSERT INTO conversations (id) 
        VALUES ($1) 
        ON CONFLICT (id) DO UPDATE SET id = EXCLUDED.id
        RETURNING created_at`

	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, convID).Scan(&conversation.CreatedAt)
	if err != nil {
		return nil, err
	}
	return conversation, nil
}

func (r *ConversationRepo) DeleteConversation(ctx context.Context, convID uuid.UUID) error {
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
		return sql.ErrNoRows
	}
	return nil
}
