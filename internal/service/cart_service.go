package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// CartService - логика корзины.
type CartService struct {
	cartRepo domain.CartRepository     // корзины в базе
	itemRepo domain.MenuItemRepository // блюда в базе
}

// NewCartService - создаёт сервис корзины.
func NewCartService(cartRepo domain.CartRepository, itemRepo domain.MenuItemRepository) *CartService {
	return &CartService{cartRepo: cartRepo, itemRepo: itemRepo}
}

// GetCart - возвращает корзину пользователя в заведении вместе с позициями.
func (s *CartService) GetCart(ctx context.Context, userID, venueID string) (*domain.Cart, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, userID, venueID)
	if err != nil {
		return nil, err
	}

	items, err := s.cartRepo.GetItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}
	cart.Items = items

	return cart, nil
}

// AddItem - кладёт блюдо в корзину, повторное добавление увеличивает количество.
func (s *CartService) AddItem(ctx context.Context, userID, venueID string, req domain.AddToCartRequest) (*domain.Cart, error) {
	if req.Quantity <= 0 {
		return nil, fmt.Errorf("%w: quantity must be > 0", domain.ErrValidation)
	}

	item, err := s.itemRepo.GetByID(ctx, req.MenuItemID)
	if err != nil {
		return nil, fmt.Errorf("menu item not found: %w", err)
	}

	if !item.IsAvailable {
		return nil, fmt.Errorf("%w: menu item is not available", domain.ErrValidation)
	}

	if item.VenueID != venueID {
		return nil, fmt.Errorf("%w: menu item does not belong to this venue", domain.ErrValidation)
	}

	cart, err := s.cartRepo.GetOrCreate(ctx, userID, venueID)
	if err != nil {
		return nil, err
	}

	if cart.VenueID != venueID {
		return nil, fmt.Errorf("%w: cart belongs to different venue, clear cart first", domain.ErrValidation)
	}

	items, err := s.cartRepo.GetItems(ctx, cart.ID)
	if err != nil {
		return nil, err
	}

	for _, ci := range items {
		if ci.MenuItemID != req.MenuItemID {
			continue
		}
		if err := s.cartRepo.UpdateItemQuantity(ctx, ci.ID, ci.Quantity+req.Quantity); err != nil {
			return nil, err
		}
		return s.GetCart(ctx, userID, venueID)
	}

	cartItem := &domain.CartItem{
		ID:         uuid.New().String(),
		CartID:     cart.ID,
		MenuItemID: req.MenuItemID,
		Quantity:   req.Quantity,
		Options:    req.Options,
	}

	if err := s.cartRepo.AddItem(ctx, cartItem); err != nil {
		return nil, err
	}

	return s.GetCart(ctx, userID, venueID)
}

// UpdateItemQuantity - меняет количество у позиции корзины.
func (s *CartService) UpdateItemQuantity(ctx context.Context, userID, itemID string, quantity int) error {
	return s.cartRepo.UpdateItemQuantity(ctx, itemID, quantity)
}

// RemoveItem - убирает позицию из корзины.
func (s *CartService) RemoveItem(ctx context.Context, userID, itemID string) error {
	return s.cartRepo.RemoveItem(ctx, itemID)
}

// Clear - очищает корзину пользователя в заведении.
func (s *CartService) Clear(ctx context.Context, userID, venueID string) error {
	cart, err := s.cartRepo.GetOrCreate(ctx, userID, venueID)
	if err != nil {
		return err
	}
	return s.cartRepo.Clear(ctx, cart.ID)
}
