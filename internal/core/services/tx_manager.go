package services

import (
	"context"
	"database/sql"
	"log/slog"
)

type txKeyType struct{}

var txKey = txKeyType{}

type TxManager struct {
	log *slog.Logger
	db  *sql.DB
}

func NewTxManager(log *slog.Logger, db *sql.DB) *TxManager {
	return &TxManager{log: log, db: db}
}

func (tm *TxManager) WithTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	tx, err := tm.db.BeginTx(ctx, nil)
	if err != nil {
		tm.log.ErrorContext(ctx, "transaction begin failed", "err", err)
		return err
	}
	ctxWithTx := context.WithValue(ctx, txKey, tx)
	if err := fn(ctxWithTx); err != nil {
		tm.log.ErrorContext(ctx, "transaction failed", "err", err)
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
