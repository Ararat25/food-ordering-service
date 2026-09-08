package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"avito-kitchen/internal/domain"
)

// ReviewRepo - отзывы в базе данных.
type ReviewRepo struct {
	db *sql.DB // соединение с базой
}

// NewReviewRepo - создаёт репозиторий отзывов.
func NewReviewRepo(db *sql.DB) *ReviewRepo {
	return &ReviewRepo{db: db}
}

// Create - сохраняет новый отзыв, дубль даёт ErrAlreadyExists.
func (r *ReviewRepo) Create(ctx context.Context, review *domain.Review) error {
	query, args, err := sb.Insert("reviews").
		Columns("id", "user_id", "venue_id", "order_id", "rating", "comment").
		Values(review.ID, review.UserID, review.VenueID, review.OrderID, review.Rating, review.Comment).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert review: %w", err)
	}

	return nil
}

// ListByVenue - возвращает отзывы заведения от новых к старым плюс общее число.
func (r *ReviewRepo) ListByVenue(ctx context.Context, venueID string, page, perPage int) ([]domain.Review, int, error) {
	countQuery, countArgs, err := sb.Select("COUNT(*)").From("reviews").
		Where(sq.Eq{"venue_id": venueID}).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count reviews: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := sb.Select(reviewColumns...).From("reviews").
		Where(sq.Eq{"venue_id": venueID}).
		OrderBy("created_at DESC").
		Limit(uint64(perPage)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list reviews: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var reviews []domain.Review
	for rows.Next() {
		var rv domain.Review
		if err := rows.Scan(&rv.ID, &rv.UserID, &rv.VenueID, &rv.OrderID, &rv.Rating, &rv.Comment, &rv.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan review: %w", err)
		}
		reviews = append(reviews, rv)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate reviews: %w", err)
	}

	return reviews, total, nil
}

// GetByUserAndOrder - находит отзыв пользователя по заказу, нет такого — ErrNotFound.
func (r *ReviewRepo) GetByUserAndOrder(ctx context.Context, userID, orderID string) (*domain.Review, error) {
	query, args, err := sb.Select(reviewColumns...).From("reviews").
		Where(sq.Eq{"user_id": userID, "order_id": orderID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var rv domain.Review
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&rv.ID, &rv.UserID, &rv.VenueID, &rv.OrderID, &rv.Rating, &rv.Comment, &rv.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}

	return &rv, nil
}
