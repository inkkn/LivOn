package postgres

import (
	"context"
	"database/sql"
)

type txKeyType struct{}

var txKey = txKeyType{}

type execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func GetExecutor(ctx context.Context, db *sql.DB) execer {
	if tx, ok := ctx.Value(txKey).(*sql.Tx); ok {
		return tx
	}
	return db
}
