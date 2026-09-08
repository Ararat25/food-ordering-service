package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"avito-kitchen/internal/domain"
)

// PushTokenRepo - пуш-токены в базе данных.
type PushTokenRepo struct {
	db *sql.DB // соединение с базой
}

// NewPushTokenRepo - создаёт репозиторий пуш-токенов.
func NewPushTokenRepo(db *sql.DB) *PushTokenRepo {
	return &PushTokenRepo{db: db}
}

// Register - обновляет платформу для своего токена, но не позволяет перехватить чужой.
func (r *PushTokenRepo) Register(ctx context.Context, t *domain.PushToken) error {
	query, args, err := sb.Insert("push_tokens").
		Columns("id", "user_id", "token", "platform").
		Values(t.ID, t.UserID, t.Token, t.Platform).
		Suffix("ON CONFLICT (token) DO UPDATE SET platform = EXCLUDED.platform WHERE push_tokens.user_id = EXCLUDED.user_id RETURNING id").
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	var id string
	err = r.db.QueryRowContext(ctx, query, args...).Scan(&id)
	if err == sql.ErrNoRows {
		return domain.ErrAlreadyExists
	}
	if err != nil {
		return fmt.Errorf("register push token: %w", err)
	}

	return nil
}

// GetByUserID - возвращает все токены пользователя.
func (r *PushTokenRepo) GetByUserID(ctx context.Context, userID string) ([]domain.PushToken, error) {
	query, args, err := sb.Select(pushTokenColumns...).From("push_tokens").
		Where(sq.Eq{"user_id": userID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list push tokens: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tokens []domain.PushToken
	for rows.Next() {
		var t domain.PushToken
		if err := rows.Scan(&t.ID, &t.UserID, &t.Token, &t.Platform, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan push token: %w", err)
		}
		tokens = append(tokens, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate push tokens: %w", err)
	}

	return tokens, nil
}

// DeleteForUser - удаляет токен только если он принадлежит указанному пользователю.
func (r *PushTokenRepo) DeleteForUser(ctx context.Context, userID, token string) error {
	query, args, err := sb.Delete("push_tokens").Where(sq.Eq{"user_id": userID, "token": token}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete push token: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted push token: %w", err)
	}
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
