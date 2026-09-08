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
func (r *CartRepo) WithTx(tx domain.Tx) domain.CartRepository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("repository: WithTx supports only *sql.Tx transactions")
	}
	return &CartRepo{db: sqlTx}
}

// GetOrCreate - находит корзину пользователя в заведении или создаёт новую.
func (r *CartRepo) GetOrCreate(ctx context.Context, userID, venueID string) (*domain.Cart, error) {
	return r.getOrCreate(ctx, userID, venueID, false)
}

// GetOrCreateForUpdate - получает корзину с row lock; вызывать внутри транзакции.
func (r *CartRepo) GetOrCreateForUpdate(ctx context.Context, userID, venueID string) (*domain.Cart, error) {
	return r.getOrCreate(ctx, userID, venueID, true)
}

func (r *CartRepo) getOrCreate(ctx context.Context, userID, venueID string, lock bool) (*domain.Cart, error) {
	var cart domain.Cart
	selectBuilder := sb.Select(cartColumns...).From("carts").Where(sq.Eq{"user_id": userID, "venue_id": venueID})
	if lock {
		selectBuilder = selectBuilder.Suffix("FOR UPDATE")
	}
	query, args, err := selectBuilder.ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&cart.ID, &cart.UserID, &cart.VenueID, &cart.CreatedAt, &cart.UpdatedAt,
	)
	if err == nil {
		return &cart, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("get cart: %w", err)
	}

	insertQuery, insertArgs, insertErr := sb.Insert("carts").
		Columns("id", "user_id", "venue_id").
		Values(uuid.New().String(), userID, venueID).
		Suffix("ON CONFLICT (user_id, venue_id) DO NOTHING").
		ToSql()
	if insertErr != nil {
		return nil, fmt.Errorf("build insert query: %w", insertErr)
	}
	if _, insertErr = r.db.ExecContext(ctx, insertQuery, insertArgs...); insertErr != nil {
		return nil, fmt.Errorf("create cart: %w", insertErr)
	}
	// При гонке создания дочитываем строку; FOR UPDATE удержит её до конца checkout.
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&cart.ID, &cart.UserID, &cart.VenueID, &cart.CreatedAt, &cart.UpdatedAt,
	)
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

// UpdateItemQuantityForUser - меняет позицию только если она принадлежит пользователю.
func (r *CartRepo) UpdateItemQuantityForUser(ctx context.Context, userID, itemID string, quantity int) error {
	query, args, err := sb.Update("cart_items").Set("quantity", quantity).
		Where("id = ? AND EXISTS (SELECT 1 FROM carts WHERE carts.id = cart_items.cart_id AND carts.user_id = ?)", itemID, userID).
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

// RemoveItemForUser - удаляет позицию только если она принадлежит пользователю.
func (r *CartRepo) RemoveItemForUser(ctx context.Context, userID, itemID string) error {
	query, args, err := sb.Delete("cart_items").
		Where("id = ? AND EXISTS (SELECT 1 FROM carts WHERE carts.id = cart_items.cart_id AND carts.user_id = ?)", itemID, userID).
		ToSql()
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
	return r.getItems(ctx, cartID, false)
}

func (r *CartRepo) GetItemsForUpdate(ctx context.Context, cartID string) ([]domain.CartItem, error) {
	return r.getItems(ctx, cartID, true)
}

func (r *CartRepo) getItems(ctx context.Context, cartID string, lock bool) ([]domain.CartItem, error) {
	builder := sb.Select(cartItemColumns...).From("cart_items").
		Where(sq.Eq{"cart_id": cartID}).
		OrderBy("created_at ASC")
	if lock {
		builder = builder.Suffix("FOR UPDATE")
	}
	query, args, err := builder.ToSql()
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
