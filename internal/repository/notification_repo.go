package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"avito-kitchen/internal/domain"
)

// NotificationRepo - уведомления в базе данных.
type NotificationRepo struct {
	db *sql.DB // соединение с базой
}

// NewNotificationRepo - создаёт репозиторий уведомлений.
func NewNotificationRepo(db *sql.DB) *NotificationRepo {
	return &NotificationRepo{db: db}
}

// Create - сохраняет новое уведомление.
func (r *NotificationRepo) Create(ctx context.Context, n *domain.Notification) error {
	query, args, err := sb.Insert("notifications").
		Columns("id", "user_id", "type", "title", "body", "data").
		Values(n.ID, n.UserID, n.Type, n.Title, n.Body, n.Data).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert notification: %w", err)
	}

	return nil
}

// ListByUser - возвращает уведомления пользователя от новых к старым плюс общее число.
func (r *NotificationRepo) ListByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Notification, int, error) {
	countQuery, countArgs, err := sb.Select("COUNT(*)").From("notifications").
		Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count notifications: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := sb.Select(notificationColumns...).From("notifications").
		Where(sq.Eq{"user_id": userID}).
		OrderBy("created_at DESC").
		Limit(uint64(perPage)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list notifications: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var notifications []domain.Notification
	for rows.Next() {
		var n domain.Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Type, &n.Title, &n.Body, &n.Data, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan notification: %w", err)
		}
		notifications = append(notifications, n)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate notifications: %w", err)
	}

	return notifications, total, nil
}

// MarkAsRead - помечает уведомление прочитанным, чужой id даёт ErrNotFound.
func (r *NotificationRepo) MarkAsRead(ctx context.Context, id string) error {
	query, args, err := sb.Update("notifications").
		Set("is_read", true).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("mark notification as read: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
