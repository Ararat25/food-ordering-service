package service

import (
	"context"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// MenuService - логика меню.
type MenuService struct {
	categoryRepo domain.MenuCategoryRepository // категории в базе
	itemRepo     domain.MenuItemRepository     // блюда в базе
}

// NewMenuService - создаёт сервис меню.
func NewMenuService(categoryRepo domain.MenuCategoryRepository, itemRepo domain.MenuItemRepository) *MenuService {
	return &MenuService{categoryRepo: categoryRepo, itemRepo: itemRepo}
}

// ListCategories - возвращает категории заведения.
func (s *MenuService) ListCategories(ctx context.Context, venueID string) ([]domain.MenuCategory, error) {
	return s.categoryRepo.ListByVenue(ctx, venueID)
}

// CreateCategory - создаёт категорию, без id придумывает новый.
func (s *MenuService) CreateCategory(ctx context.Context, category *domain.MenuCategory) error {
	if category.ID == "" {
		category.ID = uuid.New().String()
	}
	return s.categoryRepo.Create(ctx, category)
}

// UpdateCategory - обновляет категорию.
func (s *MenuService) UpdateCategory(ctx context.Context, category *domain.MenuCategory) error {
	return s.categoryRepo.Update(ctx, category)
}

// DeleteCategory - удаляет категорию.
func (s *MenuService) DeleteCategory(ctx context.Context, id string) error {
	return s.categoryRepo.Delete(ctx, id)
}

// ListItems - возвращает блюда заведения.
func (s *MenuService) ListItems(ctx context.Context, venueID string) ([]domain.MenuItem, error) {
	return s.itemRepo.ListByVenue(ctx, venueID)
}

// GetItem - возвращает блюдо по id.
func (s *MenuService) GetItem(ctx context.Context, id string) (*domain.MenuItem, error) {
	return s.itemRepo.GetByID(ctx, id)
}

// CreateItem - создаёт блюдо, без id придумывает новый.
func (s *MenuService) CreateItem(ctx context.Context, item *domain.MenuItem) error {
	if item.ID == "" {
		item.ID = uuid.New().String()
	}
	item.Options = normalizeOptions(item.Options)
	return s.itemRepo.Create(ctx, item)
}

// UpdateItem - обновляет блюдо, чужое заведение даёт ErrForbidden.
func (s *MenuService) UpdateItem(ctx context.Context, item *domain.MenuItem) error {
	existing, err := s.itemRepo.GetByID(ctx, item.ID)
	if err != nil {
		return err
	}
	if existing.VenueID != item.VenueID {
		return domain.ErrForbidden
	}
	item.Options = normalizeOptions(item.Options)
	return s.itemRepo.Update(ctx, item)
}

// DeleteItem - удаляет блюдо, чужое заведение даёт ErrForbidden.
func (s *MenuService) DeleteItem(ctx context.Context, venueID, id string) error {
	existing, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.VenueID != venueID {
		return domain.ErrForbidden
	}
	return s.itemRepo.Delete(ctx, id)
}

// UpdateAvailability - меняет наличие блюда, чужое заведение даёт ErrForbidden.
func (s *MenuService) UpdateAvailability(ctx context.Context, venueID, id string, available bool) error {
	existing, err := s.itemRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if existing.VenueID != venueID {
		return domain.ErrForbidden
	}
	return s.itemRepo.UpdateAvailability(ctx, id, available)
}
