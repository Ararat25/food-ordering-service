package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"

	"avito-kitchen/internal/domain"
)

// MenuCategoryRepo - категории меню в базе данных.
type MenuCategoryRepo struct {
	db *sql.DB // соединение с базой
}

// NewMenuCategoryRepo - создаёт репозиторий категорий.
func NewMenuCategoryRepo(db *sql.DB) *MenuCategoryRepo {
	return &MenuCategoryRepo{db: db}
}

// Create - сохраняет новую категорию, дубль даёт ErrAlreadyExists.
func (r *MenuCategoryRepo) Create(ctx context.Context, cat *domain.MenuCategory) error {
	query, args, err := sb.Insert("menu_categories").
		Columns("id", "venue_id", "name", "sort_order").
		Values(cat.ID, cat.VenueID, cat.Name, cat.SortOrder).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			return domain.ErrAlreadyExists
		}
		return fmt.Errorf("insert category: %w", err)
	}

	return nil
}

// GetByID - находит категорию по id.
func (r *MenuCategoryRepo) GetByID(ctx context.Context, id string) (*domain.MenuCategory, error) {
	query, args, err := sb.Select(menuCategoryColumns...).From("menu_categories").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var cat domain.MenuCategory
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&cat.ID, &cat.VenueID, &cat.Name, &cat.SortOrder, &cat.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get category: %w", err)
	}

	return &cat, nil
}

// ListByVenue - возвращает категории заведения по порядку показа плюс общее число.
func (r *MenuCategoryRepo) ListByVenue(ctx context.Context, venueID string, page, perPage int) ([]domain.MenuCategory, int, error) {
	countQuery, countArgs, err := sb.Select("COUNT(*)").From("menu_categories").
		Where(sq.Eq{"venue_id": venueID}).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count categories: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := sb.Select(menuCategoryColumns...).From("menu_categories").
		Where(sq.Eq{"venue_id": venueID}).
		OrderBy("sort_order ASC").
		Limit(uint64(perPage)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list categories: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var categories []domain.MenuCategory
	for rows.Next() {
		var cat domain.MenuCategory
		if err := rows.Scan(&cat.ID, &cat.VenueID, &cat.Name, &cat.SortOrder, &cat.CreatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan category: %w", err)
		}
		categories = append(categories, cat)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate categories: %w", err)
	}

	return categories, total, nil
}

// Update - обновляет категорию, чужой id даёт ErrNotFound.
func (r *MenuCategoryRepo) Update(ctx context.Context, cat *domain.MenuCategory) error {
	query, args, err := sb.Update("menu_categories").
		Set("name", cat.Name).
		Set("sort_order", cat.SortOrder).
		Where(sq.Eq{"id": cat.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete - удаляет категорию, чужой id даёт ErrNotFound.
func (r *MenuCategoryRepo) Delete(ctx context.Context, id string) error {
	query, args, err := sb.Delete("menu_categories").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete category: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// MenuItemRepo - блюда в базе данных.
type MenuItemRepo struct {
	db DBTX // соединение или транзакция
}

// NewMenuItemRepo - создаёт репозиторий блюд.
func NewMenuItemRepo(db *sql.DB) *MenuItemRepo {
	return &MenuItemRepo{db: db}
}

// MenuItemRepo обещает выполнять контракт MenuItemRepository.
var _ domain.MenuItemRepository = (*MenuItemRepo)(nil)

// WithTx - возвращает копию репозитория, пишущую в заданную транзакцию.
func (r *MenuItemRepo) WithTx(tx domain.Tx) domain.MenuItemRepository {
	sqlTx, ok := tx.(*sql.Tx)
	if !ok {
		panic("repository: WithTx supports only *sql.Tx transactions")
	}
	return &MenuItemRepo{db: sqlTx}
}

// Create - сохраняет новое блюдо.
func (r *MenuItemRepo) Create(ctx context.Context, item *domain.MenuItem) error {
	query, args, err := sb.Insert("menu_items").
		Columns("id", "category_id", "venue_id", "name", "description", "price", "image_url", "is_available", "options").
		Values(item.ID, item.CategoryID, item.VenueID, item.Name, item.Description, item.Price, item.ImageURL, item.IsAvailable, item.Options).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("insert menu item: %w", err)
	}

	return nil
}

// GetByID - находит блюдо по id.
func (r *MenuItemRepo) GetByID(ctx context.Context, id string) (*domain.MenuItem, error) {
	query, args, err := sb.Select(menuItemColumns...).From("menu_items").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	var item domain.MenuItem
	err = r.db.QueryRowContext(ctx, query, args...).Scan(
		&item.ID, &item.CategoryID, &item.VenueID, &item.Name, &item.Description,
		&item.Price, &item.ImageURL, &item.IsAvailable, &item.Options,
		&item.CreatedAt, &item.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get menu item: %w", err)
	}

	return &item, nil
}

// GetByIDs - находит пачку блюд по id одним запросом, для сборки заказа без N+1.
func (r *MenuItemRepo) GetByIDs(ctx context.Context, ids []string) ([]domain.MenuItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sb.Select(menuItemColumns...).From("menu_items").
		Where(sq.Eq{"id": ids}).
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list menu items by ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.VenueID, &item.Name, &item.Description,
			&item.Price, &item.ImageURL, &item.IsAvailable, &item.Options,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan menu item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate menu items by ids: %w", err)
	}

	return items, nil
}

// GetByIDsForUpdate - та же пачка, но с локой строк: между проверкой наличия
// и созданием заказа блюдо никто не снимет с продажи. Вне транзакции лока
// тут же отпускается, так что метод имеет смысл только на WithTx-копии.
func (r *MenuItemRepo) GetByIDsForUpdate(ctx context.Context, ids []string) ([]domain.MenuItem, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	query, args, err := sb.Select(menuItemColumns...).From("menu_items").
		Where(sq.Eq{"id": ids}).
		Suffix("FOR UPDATE").
		ToSql()
	if err != nil {
		return nil, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list menu items by ids: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.VenueID, &item.Name, &item.Description,
			&item.Price, &item.ImageURL, &item.IsAvailable, &item.Options,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan menu item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate menu items by ids: %w", err)
	}

	return items, nil
}

// ListByVenue - возвращает блюда заведения с фильтрами плюс общее число.
func (r *MenuItemRepo) ListByVenue(ctx context.Context, filters domain.MenuItemFilters) ([]domain.MenuItem, int, error) {
	baseQuery := sb.Select(menuItemColumns...).From("menu_items").Where(sq.Eq{"venue_id": filters.VenueID})
	countBaseQuery := sb.Select("COUNT(*)").From("menu_items").Where(sq.Eq{"venue_id": filters.VenueID})

	if filters.Search != "" {
		search := "%" + strings.ToLower(filters.Search) + "%"
		condition := sq.Or{
			sq.Like{"LOWER(name)": search},
			sq.Like{"LOWER(description)": search},
		}
		baseQuery = baseQuery.Where(condition)
		countBaseQuery = countBaseQuery.Where(condition)
	}

	if filters.Available != nil {
		baseQuery = baseQuery.Where(sq.Eq{"is_available": *filters.Available})
		countBaseQuery = countBaseQuery.Where(sq.Eq{"is_available": *filters.Available})
	}

	countQuery, countArgs, err := countBaseQuery.ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count menu items: %w", err)
	}

	offset := (filters.Page - 1) * filters.PerPage
	query, args, err := baseQuery.
		OrderBy("name ASC").
		Limit(uint64(filters.PerPage)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list menu items: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.VenueID, &item.Name, &item.Description,
			&item.Price, &item.ImageURL, &item.IsAvailable, &item.Options,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan menu item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate menu items: %w", err)
	}

	return items, total, nil
}

// ListByCategory - возвращает блюда одной категории по алфавиту плюс общее число.
func (r *MenuItemRepo) ListByCategory(ctx context.Context, categoryID string, page, perPage int) ([]domain.MenuItem, int, error) {
	countQuery, countArgs, err := sb.Select("COUNT(*)").From("menu_items").
		Where(sq.Eq{"category_id": categoryID}).ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build count query: %w", err)
	}

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count menu items by category: %w", err)
	}

	offset := (page - 1) * perPage
	query, args, err := sb.Select(menuItemColumns...).From("menu_items").
		Where(sq.Eq{"category_id": categoryID}).
		OrderBy("name ASC").
		Limit(uint64(perPage)).
		Offset(uint64(offset)).
		ToSql()
	if err != nil {
		return nil, 0, fmt.Errorf("build query: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list menu items by category: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var items []domain.MenuItem
	for rows.Next() {
		var item domain.MenuItem
		if err := rows.Scan(
			&item.ID, &item.CategoryID, &item.VenueID, &item.Name, &item.Description,
			&item.Price, &item.ImageURL, &item.IsAvailable, &item.Options,
			&item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan menu item: %w", err)
		}
		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate menu items by category: %w", err)
	}

	return items, total, nil
}

// Update - обновляет блюдо, чужой id даёт ErrNotFound.
func (r *MenuItemRepo) Update(ctx context.Context, item *domain.MenuItem) error {
	query, args, err := sb.Update("menu_items").
		Set("name", item.Name).
		Set("description", item.Description).
		Set("price", item.Price).
		Set("image_url", item.ImageURL).
		Set("is_available", item.IsAvailable).
		Set("options", item.Options).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": item.ID}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update menu item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// Delete - удаляет блюдо, чужой id даёт ErrNotFound.
func (r *MenuItemRepo) Delete(ctx context.Context, id string) error {
	query, args, err := sb.Delete("menu_items").Where(sq.Eq{"id": id}).ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("delete menu item: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

// UpdateAvailability - меняет наличие блюда, чужой id даёт ErrNotFound.
func (r *MenuItemRepo) UpdateAvailability(ctx context.Context, id string, available bool) error {
	query, args, err := sb.Update("menu_items").
		Set("is_available", available).
		Set("updated_at", sq.Expr("NOW()")).
		Where(sq.Eq{"id": id}).
		ToSql()
	if err != nil {
		return fmt.Errorf("build query: %w", err)
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update availability: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return domain.ErrNotFound
	}

	return nil
}
