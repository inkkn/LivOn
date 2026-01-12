package postgres

import (
	"context"
	"database/sql"
	"livon/internal/core/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

/*
	type UserRepository interface {
		GetUserByID(ctx context.Context, id string) (*User, error)
		CreateUser(ctx context.Context, id string) (*User, error)
		DeleteUser(ctx context.Context, id string) error
	}
*/

func (r *UserRepo) GetUserByID(ctx context.Context, phone string) (*domain.User, error) {
	if phone == "" {
		return nil, domain.ErrInvalidUserID
	}
	user := &domain.User{ID: phone}
	query := `SELECT created_at FROM users WHERE id = $1`
	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, phone).Scan(&user.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, domain.ErrUserNotFound
		}
		return nil, err
	}
	return user, nil
}

func (r *UserRepo) CreateUser(ctx context.Context, phone string) (*domain.User, error) {
	if phone == "" {
		return nil, domain.ErrInvalidUserID
	}
	user := &domain.User{
		ID: phone,
	}
	// Insert new user or do nothing if phone already exists
	// We return the created_at to populate our core model
	query :=
		`INSERT INTO users (id) 
        VALUES ($1) 
        ON CONFLICT (id) DO NOTHING
        RETURNING created_at`

	exec := GetExecutor(ctx, r.db)
	err := exec.QueryRowContext(ctx, query, phone).Scan(&user.CreatedAt)
	switch {
	case err == nil:
		// Created
		return user, nil

	case err == sql.ErrNoRows:
		// Already exists
		return r.GetUserByID(ctx, phone)

	default:
		return nil, err
	}
}

func (r *UserRepo) DeleteUser(ctx context.Context, phone string) error {
	if phone == "" {
		return domain.ErrInvalidUserID
	}
	query := `DELETE FROM users WHERE id = $1`
	exec := GetExecutor(ctx, r.db)
	result, err := exec.ExecContext(ctx, query, phone)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}
