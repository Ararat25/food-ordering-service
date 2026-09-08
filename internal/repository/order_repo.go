package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// OrderRepo - заказы в базе данных.
type OrderRepo struct {
	db DBTX // соединение или транзакция
}

// OrderRepo обещает выполнять контракт OrderRepository.
var _ domain.OrderRepository = (*OrderRepo)(nil)

// NewOrderRepo - создаёт репозиторий заказов.
func NewOrderRepo(db *sql.DB) *OrderRepo {
	return &OrderRepo{db: db}
}

// WithTx - возвращает копию репозитория, пишущую в заданную транзакцию.
func (r *OrderRepo) WithTx(tx domain.Tx) domain.OrderRepository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("repository: WithTx supports only *sql.Tx transactions")
	}
	return &OrderRepo{db: sqlTx}
}

// Create - сохраняет заказ без позиций.
func (r *OrderRepo) Create(ctx context.Context, order *domain.Order) error {
	query, args, err := sb.Insert("orders").
		Columns("id", "user_id", "venue_id", "status", "delivery_type",
			"delivery_address", "delivery_time", "estimated_delivery_time",
			"comment", "total_amount").
		Values(order.ID, order.UserID, order.VenueID, order.Status, order.DeliveryType,
			order.DeliveryAddress, order.DeliveryTime, order.EstimatedDeliveryTime,
			order.Comment, order.TotalAmount).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	return nil
}

// CreateWithItems - сохраняет заказ с позициями и платежом.
func (r *OrderRepo) CreateWithItems(ctx context.Context, order *domain.Order, items []domain.OrderItem, paymentMethod string) error {
	db, ok := r.db.(*sql.DB)
	if !ok {
		return r.insertOrderWithItems(ctx, r.db, order, items, paymentMethod)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := r.insertOrderWithItems(ctx, tx, order, items, paymentMethod); err != nil {
		return err
	}

	return tx.Commit()
}

// idempotencyValue - пустой ключ пишет как NULL, иначе ” конфликтовал бы в уникальном индексе.
func idempotencyValue(key string) any {
	if key == "" {
		return nil
	}
	return key
}

// insertOrderWithItems - пишет заказ, позиции и платёж в заданный DBTX.
func (r *OrderRepo) insertOrderWithItems(ctx context.Context, q DBTX, order *domain.Order, items []domain.OrderItem, paymentMethod string) error {
	orderQuery, orderArgs, err := sb.Insert("orders").
		Columns("id", "user_id", "venue_id", "status", "delivery_type",
			"delivery_address", "delivery_time", "estimated_delivery_time",
			"comment", "total_amount", "idempotency_key").
		Values(order.ID, order.UserID, order.VenueID, order.Status, order.DeliveryType,
			order.DeliveryAddress, order.DeliveryTime, order.EstimatedDeliveryTime,
			order.Comment, order.TotalAmount, idempotencyValue(order.IdempotencyKey)).
		Suffix("RETURNING created_at, updated_at").
		ToSql()
	if err != nil {
		return fmt.Errorf("build order query: %w", err)
	}

	if err := q.QueryRowContext(ctx, orderQuery, orderArgs...).Scan(&order.CreatedAt, &order.UpdatedAt); err != nil {
		return fmt.Errorf("insert order: %w", err)
	}

	for i := range items {
		item := &items[i]
		itemQuery, itemArgs, err := sb.Insert("order_items").
			Columns("id", "order_id", "menu_item_id", "quantity", "price", "options").
			Values(item.ID, item.OrderID, item.MenuItemID, item.Quantity, item.Price, item.Options).
			Suffix("RETURNING created_at").
			ToSql()
		if err != nil {
			return fmt.Errorf("build item query: %w", err)
		}

		if err := q.QueryRowContext(ctx, itemQuery, itemArgs...).Scan(&item.CreatedAt); err != nil {
			return fmt.Errorf("insert order item: %w", err)
		}
	}

	paymentQuery, paymentArgs, err := sb.Insert("payments").
		Columns("id", "order_id", "status", "amount", "method").
		Values(uuid.New().String(), order.ID, domain.PaymentStatusPending, order.TotalAmount, paymentMethod).
		ToSql()
	if err != nil {
		return fmt.Errorf("build payment query: %w", err)
	}

	if _, err := q.ExecContext(ctx, paymentQuery, paymentArgs...); err != nil {
		return fmt.Errorf("insert payment: %w", err)
	}

	return nil
}

// rowScanner - общее для *sql.Row и *sql.Rows: умеет Scan.
type rowScanner interface {
	Scan(dest ...any) error
}

// scanOrder - читает заказ из строки, NULL-ключ идемпотентности превращает в пустую строку.
func scanOrder(s rowScanner, order *domain.Order) error {
	var key sql.NullString
	err := s.Scan(
		&order.ID, &order.UserID, &order.VenueID, &order.Status, &order.DeliveryType,
		&order.DeliveryAddress, &order.DeliveryTime, &order.EstimatedDeliveryTime,
		&order.Comment, &order.TotalAmount, &order.CreatedAt, &order.UpdatedAt, &key,
	)
	if err != nil {
		return err
	}
	order.IdempotencyKey = key.String
	return nil
}

// GetByUserAndIdempotencyKey - находит заказ пользователя по ключу идемпотентности.
func (r *OrderRepo) GetByUserAndIdempotencyKey(ctx context.Context, userID, key string) (*domain.Order, error) {
	query, args, err := sb.Select(orderColumns...).From("orders").Where(sq.Eq{"user_id": userID, "idempotency_key": key}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var order domain.Order
	err = scanOrder(r.db.QueryRowContext(ctx, query, args...), &order)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get order by user and idempotency key: %w", err)
	}

	return &order, nil
}

// GetByID - находит заказ по id.
func (r *OrderRepo) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	return r.getByIDWhere(ctx, sq.Eq{"id": id})
}

// GetByIDForUser - возвращает заказ только его владельцу.
func (r *OrderRepo) GetByIDForUser(ctx context.Context, id, userID string) (*domain.Order, error) {
	return r.getByIDWhere(ctx, sq.Eq{"id": id, "user_id": userID})
}

// GetByIDForVenue - возвращает заказ только заведению-владельцу.
func (r *OrderRepo) GetByIDForVenue(ctx context.Context, id, venueID string) (*domain.Order, error) {
	return r.getByIDWhere(ctx, sq.Eq{"id": id, "venue_id": venueID})
}

func (r *OrderRepo) getByIDWhere(ctx context.Context, where sq.Eq) (*domain.Order, error) {
	query, args, err := sb.Select(orderColumns...).From("orders").Where(where).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var order domain.Order
	err = scanOrder(r.db.QueryRowContext(ctx, query, args...), &order)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get scoped order: %w", err)
	}

	return &order, nil
}

// ListByUser - возвращает заказы пользователя от новых к старым плюс общее число.
func (r *OrderRepo) ListByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Order, int, error) {
	countQuery, countArgs, err := sb.Select("COUNT(*)").From("orders").
		Where(sq.Eq{"user_id": userID}).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := sb.Select(orderColumns...).From("orders").
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
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := scanOrder(rows, &o); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders (user): %w", err)
	}

	return orders, total, nil
}

// ListByVenue - возвращает заказы заведения с фильтром по статусу плюс общее число.
func (r *OrderRepo) ListByVenue(ctx context.Context, venueID string, status string, page, perPage int) ([]domain.Order, int, error) {
	baseQuery := sb.Select(orderColumns...).From("orders").Where(sq.Eq{"venue_id": venueID})
	countBaseQuery := sb.Select("COUNT(*)").From("orders").Where(sq.Eq{"venue_id": venueID})

	if status != "" {
		baseQuery = baseQuery.Where(sq.Eq{"status": status})
		countBaseQuery = countBaseQuery.Where(sq.Eq{"status": status})
	}

	countQuery, countArgs, err := countBaseQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count orders: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := baseQuery.OrderBy("created_at DESC").
		Limit(uint64(perPage)).Offset(uint64(offset)).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list orders: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var orders []domain.Order
	for rows.Next() {
		var o domain.Order
		if err := scanOrder(rows, &o); err != nil {
			return nil, 0, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, o)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate orders (venue): %w", err)
	}

	return orders, total, nil
}

// UpdateStatusFrom - меняет статус только если в БД ещё находится ожидаемое состояние.
func (r *OrderRepo) UpdateStatusFrom(ctx context.Context, id string, from, status domain.OrderStatus) error {
	builder := sb.Update("orders").
		Set("status", status).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id, "status": from})
	if status == domain.OrderStatusDelivered {
		builder = builder.Set("delivery_time", sq.Expr("NOW()"))
	}
	query, args, err := builder.ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrConflict
	}

	return nil
}

// GetItems - возвращает позиции заказа.
func (r *OrderRepo) GetItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	query, args, err := sb.Select(orderItemColumns...).From("order_items").
		Where(sq.Eq{"order_id": orderID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list order items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.OrderItem
	for rows.Next() {
		var item domain.OrderItem
		if err := rows.Scan(&item.ID, &item.OrderID, &item.MenuItemID, &item.Quantity, &item.Price, &item.Options, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan order item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate order items: %w", err)
	}

	return items, nil
}
