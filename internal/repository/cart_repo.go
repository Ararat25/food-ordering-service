package repository

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// CartRepo - корзины в базе данных.
type CartRepo struct {
	db DBTX // соединение или транзакция
}

// CartRepo обещает выполнять контракт CartRepository.
var _ domain.CartRepository = (*CartRepo)(nil)

// NewCartRepo - создаёт репозиторий корзин.
func NewCartRepo(db *sql.DB) *CartRepo {
	return &CartRepo{db: db}
}

// WithTx - возвращает копию репозитория, пишущую в заданную транзакцию.
func (r *CartRepo) WithTx(tx *sql.Tx) domain.CartRepository {
	return &CartRepo{db: tx}
}

// GetOrCreate - находит корзину пользователя в заведении или создаёт новую.
func (r *CartRepo) GetOrCreate(ctx context.Context, userID, venueID string) (*domain.Cart, error) {
	var cart domain.Cart

	query, args, err := sb.Select(cartColumns...).From("carts").
		Where(sq.Eq{"user_id": userID, "venue_id": venueID}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&cart.ID, &cart.UserID, &cart.VenueID, &cart.CreatedAt, &cart.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		cartID := uuid.New().String()
		insertQuery, insertArgs, insertErr := sb.Insert("carts").
			Columns("id", "user_id", "venue_id").
			Values(cartID, userID, venueID).
			ToSql()
		if insertErr != nil {
			return nil, fmt.Errorf("build insert query: %w", insertErr)
		}

		_, insertErr = r.db.ExecContext(ctx, insertQuery, insertArgs...)
		if insertErr != nil {
			return nil, fmt.Errorf("create cart: %w", insertErr)
		}

		cart.ID = cartID
		cart.UserID = userID
		cart.VenueID = venueID
		return &cart, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	return &cart, nil
}

// GetByID - находит корзину по id.
func (r *CartRepo) GetByID(ctx context.Context, id string) (*domain.Cart, error) {
	query, args, err := sb.Select(cartColumns...).From("carts").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var cart domain.Cart
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&cart.ID, &cart.UserID, &cart.VenueID, &cart.CreatedAt, &cart.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	return &cart, nil
}

// Clear - убирает все позиции из корзины.
func (r *CartRepo) Clear(ctx context.Context, cartID string) error {
	query, args, err := sb.Delete("cart_items").Where(sq.Eq{"cart_id": cartID}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}

	return nil
}

// AddItem - добавляет позицию в корзину.
func (r *CartRepo) AddItem(ctx context.Context, item *domain.CartItem) error {
	query, args, err := sb.Insert("cart_items").
		Columns("id", "cart_id", "menu_item_id", "quantity", "options").
		Values(item.ID, item.CartID, item.MenuItemID, item.Quantity, item.Options).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("add cart item: %w", err)
	}

	return nil
}

// UpdateItemQuantity - меняет количество у позиции, чужой id даёт ErrNotFound.
func (r *CartRepo) UpdateItemQuantity(ctx context.Context, itemID string, quantity int) error {
	query, args, err := sb.Update("cart_items").
		Set("quantity", quantity).
		Where(sq.Eq{"id": itemID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update cart item quantity: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// RemoveItem - убирает позицию из корзины, чужой id даёт ErrNotFound.
func (r *CartRepo) RemoveItem(ctx context.Context, itemID string) error {
	query, args, err := sb.Delete("cart_items").Where(sq.Eq{"id": itemID}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("remove cart item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// GetItems - возвращает все позиции корзины по дате добавления.
func (r *CartRepo) GetItems(ctx context.Context, cartID string) ([]domain.CartItem, error) {
	query, args, err := sb.Select(cartItemColumns...).From("cart_items").
		Where(sq.Eq{"cart_id": cartID}).
		OrderBy("created_at ASC").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list cart items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.CartItem
	for rows.Next() {
		var item domain.CartItem
		if err := rows.Scan(&item.ID, &item.CartID, &item.MenuItemID, &item.Quantity, &item.Options, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan cart item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cart items: %w", err)
	}

	return items, nil
}
