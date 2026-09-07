package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// OrderService - логика заказов.
type OrderService struct {
	orderRepo domain.OrderRepository    // заказы в базе
	cartRepo  domain.CartRepository     // корзины в базе
	itemRepo  domain.MenuItemRepository // блюда в базе
}

// NewOrderService - создаёт сервис заказов.
func NewOrderService(orderRepo domain.OrderRepository, cartRepo domain.CartRepository, itemRepo domain.MenuItemRepository) *OrderService {
	return &OrderService{orderRepo: orderRepo, cartRepo: cartRepo, itemRepo: itemRepo}
}

// Create - делает заказ из корзины: проверяет блюда, считает сумму, чистит корзину.
func (s *OrderService) Create(ctx context.Context, userID string, req domain.CreateOrderRequest) (*domain.Order, error) {
	if req.VenueID == "" {
		return nil, fmt.Errorf("%w: venue_id is required", domain.ErrValidation)
	}
	if req.DeliveryType != "delivery" && req.DeliveryType != "pickup" {
		return nil, fmt.Errorf("%w: invalid delivery type: %s", domain.ErrValidation, req.DeliveryType)
	}

	cart, err := s.cartRepo.GetOrCreate(ctx, userID, req.VenueID)
	if err != nil {
		return nil, fmt.Errorf("cart not found: %w", err)
	}

	cartItems, err := s.cartRepo.GetItems(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	if len(cartItems) == 0 {
		return nil, fmt.Errorf("%w: cart is empty", domain.ErrValidation)
	}

	var totalAmount float64
	var orderItems []domain.OrderItem
	orderID := uuid.New().String()

	for _, ci := range cartItems {
		menuItem, err := s.itemRepo.GetByID(ctx, ci.MenuItemID)
		if err != nil {
			return nil, fmt.Errorf("menu item %s not found: %w", ci.MenuItemID, err)
		}

		if !menuItem.IsAvailable {
			return nil, fmt.Errorf("%w: menu item %s is not available", domain.ErrValidation, menuItem.Name)
		}

		itemTotal := menuItem.Price * float64(ci.Quantity)
		totalAmount += itemTotal

		orderItem := domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    orderID,
			MenuItemID: ci.MenuItemID,
			Quantity:   ci.Quantity,
			Price:      menuItem.Price,
			Options:    ci.Options,
		}
		orderItems = append(orderItems, orderItem)
	}

	order := &domain.Order{
		ID:              orderID,
		UserID:          userID,
		VenueID:         cart.VenueID,
		Status:          domain.OrderStatusPending,
		DeliveryType:    domain.DeliveryType(req.DeliveryType),
		DeliveryAddress: req.DeliveryAddress,
		Comment:         req.Comment,
		TotalAmount:     totalAmount,
	}

	if err := s.orderRepo.CreateWithItems(ctx, order, orderItems); err != nil {
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := s.cartRepo.Clear(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("failed to clear cart: %w", err)
	}

	return order, nil
}

// GetByID - возвращает заказ по id.
func (s *OrderService) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// ListByUser - возвращает заказы пользователя постранично.
func (s *OrderService) ListByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Order, int, error) {
	page, perPage = domain.NormalizePagination(page, perPage)
	return s.orderRepo.ListByUser(ctx, userID, page, perPage)
}

// ListByVenue - возвращает заказы заведения постранично.
func (s *OrderService) ListByVenue(ctx context.Context, venueID string, status string, page, perPage int) ([]domain.Order, int, error) {
	page, perPage = domain.NormalizePagination(page, perPage)
	return s.orderRepo.ListByVenue(ctx, venueID, status, page, perPage)
}

// UpdateStatus - меняет статус заказа, если такой переход разрешён.
func (s *OrderService) UpdateStatus(ctx context.Context, id string, status domain.OrderStatus) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if err := validateTransition(order.Status, status); err != nil {
		return err
	}

	return s.orderRepo.UpdateStatus(ctx, id, status)
}

// UpdateVenueOrderStatus - меняет статус, но только своего заказа, чужой даёт ErrForbidden.
func (s *OrderService) UpdateVenueOrderStatus(ctx context.Context, venueID, id string, status domain.OrderStatus) error {
	order, err := s.orderRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if order.VenueID != venueID {
		return domain.ErrForbidden
	}

	if err := validateTransition(order.Status, status); err != nil {
		return err
	}

	return s.orderRepo.UpdateStatus(ctx, id, status)
}

// validateTransition - проверяет, разрешён ли переход из статуса from в статус to.
func validateTransition(from, to domain.OrderStatus) error {
	validTransitions := map[domain.OrderStatus][]domain.OrderStatus{
		domain.OrderStatusPending:    {domain.OrderStatusAccepted, domain.OrderStatusCancelled},
		domain.OrderStatusAccepted:   {domain.OrderStatusPreparing, domain.OrderStatusCancelled},
		domain.OrderStatusPreparing:  {domain.OrderStatusReady},
		domain.OrderStatusReady:      {domain.OrderStatusDelivering, domain.OrderStatusDelivered},
		domain.OrderStatusDelivering: {domain.OrderStatusDelivered},
	}

	for _, s := range validTransitions[from] {
		if s == to {
			return nil
		}
	}

	return fmt.Errorf("invalid status transition from %s to %s", from, to)
}

// GetItems - возвращает позиции заказа.
func (s *OrderService) GetItems(ctx context.Context, orderID string) ([]domain.OrderItem, error) {
	return s.orderRepo.GetItems(ctx, orderID)
}

// Repeat - повторяет заказ заново, недоступные блюда молча пропускает.
func (s *OrderService) Repeat(ctx context.Context, userID, orderID string) (*domain.Order, error) {
	original, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, err
	}

	if original.UserID != userID {
		return nil, domain.ErrForbidden
	}

	items, err := s.orderRepo.GetItems(ctx, orderID)
	if err != nil {
		return nil, err
	}

	var totalAmount float64
	var newItems []domain.OrderItem
	newOrderID := uuid.New().String()

	for _, item := range items {
		menuItem, err := s.itemRepo.GetByID(ctx, item.MenuItemID)
		if err != nil {
			continue
		}

		if !menuItem.IsAvailable {
			continue
		}

		itemTotal := menuItem.Price * float64(item.Quantity)
		totalAmount += itemTotal

		newItem := domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    newOrderID,
			MenuItemID: item.MenuItemID,
			Quantity:   item.Quantity,
			Price:      menuItem.Price,
			Options:    item.Options,
		}
		newItems = append(newItems, newItem)
	}

	if len(newItems) == 0 {
		return nil, fmt.Errorf("no available items to repeat")
	}

	order := &domain.Order{
		ID:              newOrderID,
		UserID:          userID,
		VenueID:         original.VenueID,
		Status:          domain.OrderStatusPending,
		DeliveryType:    original.DeliveryType,
		DeliveryAddress: original.DeliveryAddress,
		Comment:         original.Comment,
		TotalAmount:     totalAmount,
	}

	if err := s.orderRepo.CreateWithItems(ctx, order, newItems); err != nil {
		return nil, err
	}

	return order, nil
}
