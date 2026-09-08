package repository

import (
	"context"
	"database/sql"
	"fmt"

	"avito-kitchen/internal/domain"
)

// UserRepo checks whether a demo user identity exists before the API reaches FK-backed writes.
type UserRepo struct {
	db *sql.DB
}

var _ domain.UserIdentityRepository = (*UserRepo)(nil)

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Exists(ctx context.Context, id string) (bool, error) {
	var exists bool
	if err := r.db.QueryRowContext(ctx, `SELECT EXISTS (SELECT 1 FROM users WHERE id = $1)`, id).Scan(&exists); err != nil {
		return false, fmt.Errorf("check user identity: %w", err)
	}
	return exists, nil
}
