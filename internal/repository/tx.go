package repository

import (
	"context"
	"database/sql"

	"avito-kitchen/internal/domain"
)

type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type Transactor struct {
	db *sql.DB // пул соединений
}

func NewTransactor(db *sql.DB) *Transactor {
	return &Transactor{db: db}
}

func (t *Transactor) BeginTx(ctx context.Context) (domain.Tx, error) {
	return t.db.BeginTx(ctx, nil)
}
