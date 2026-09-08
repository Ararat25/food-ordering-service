package repository

import (
	"context"
	"database/sql"
	"fmt"

	"avito-kitchen/internal/domain"
)

// UserRepo проверяет, существует ли демонстрационный идентификатор пользователя, прежде чем API перейдет к записям с поддержкой FK.
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
