package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"

	"avito-kitchen/internal/domain"
)

// PaymentRepo - оплаты в базе данных.
type PaymentRepo struct {
	db *sql.DB // соединение с базой
}

// NewPaymentRepo - создаёт репозиторий оплат.
func NewPaymentRepo(db *sql.DB) *PaymentRepo {
	return &PaymentRepo{db: db}
}

// Create - сохраняет новую оплату.
func (r *PaymentRepo) Create(ctx context.Context, p *domain.Payment) error {
	query, args, err := sb.Insert("payments").
		Columns("id", "order_id", "status", "amount", "method").
		Values(p.ID, p.OrderID, p.Status, p.Amount, p.Method).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

// GetByOrderID - находит оплату по заказу.
func (r *PaymentRepo) GetByOrderID(ctx context.Context, orderID string) (*domain.Payment, error) {
	query, args, err := sb.Select(paymentColumns...).From("payments").Where(sq.Eq{"order_id": orderID}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var p domain.Payment
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&p.ID, &p.OrderID, &p.Status, &p.Amount, &p.Method, &p.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get payment: %w", err)
	}

	return &p, nil
}

// UpdateStatus - меняет статус оплаты, чужой id даёт ErrNotFound.
func (r *PaymentRepo) UpdateStatus(ctx context.Context, id string, status domain.PaymentStatus) error {
	query, args, err := sb.Update("payments").
		Set("status", status).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update payment status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
