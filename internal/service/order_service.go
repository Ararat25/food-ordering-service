package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// defaultPaymentMethod - способ оплаты, если клиент его не указал.
const defaultPaymentMethod = "card"

// OrderService - логика заказов.
type OrderService struct {
	orderRepo  domain.OrderRepository    // заказы в базе
	cartRepo   domain.CartRepository     // корзины в базе
	itemRepo   domain.MenuItemRepository // блюда в базе
	transactor domain.Transactor         // открывает транзакции
}

// NewOrderService - создаёт сервис заказов.
func NewOrderService(orderRepo domain.OrderRepository, cartRepo domain.CartRepository, itemRepo domain.MenuItemRepository, transactor domain.Transactor) *OrderService {
	return &OrderService{orderRepo: orderRepo, cartRepo: cartRepo, itemRepo: itemRepo, transactor: transactor}
}

// Create - делает заказ из корзины: проверяет блюда, считает сумму, чистит корзину.
func (s *OrderService) Create(ctx context.Context, userID string, req domain.CreateOrderRequest) (*domain.Order, error) {
	paymentMethod, err := checkCreateRequest(req)
	if err != nil {
		return nil, err
	}

	// Повтор с тем же ключом возвращает уже созданный заказ, а не дубль.
	if existing, done, err := s.findIdempotentOrder(ctx, userID, req.IdempotencyKey); err != nil || done {
		return existing, err
	}

	tx, err := s.transactor.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin order tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	orderTxRepo := s.orderRepo.WithTx(tx)
	cartTxRepo := s.cartRepo.WithTx(tx)
	itemTxRepo := s.itemRepo.WithTx(tx)
	cart, err := cartTxRepo.GetOrCreateForUpdate(ctx, userID, req.VenueID)
	if err != nil {
		return nil, fmt.Errorf("lock cart: %w", err)
	}
	if existing, done, idemErr := s.findIdempotentOrder(ctx, userID, req.IdempotencyKey); idemErr != nil || done {
		return existing, idemErr
	}
	cartItems, err := cartTxRepo.GetItemsForUpdate(ctx, cart.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart items: %w", err)
	}
	if len(cartItems) == 0 {
		return nil, fmt.Errorf("%w: cart is empty", domain.ErrValidation)
	}

	// Блюда грузим с локой строк: между проверкой наличия и записью
	// заказа их никто не снимет с продажи.
	ids := make([]string, 0, len(cartItems))
	for _, ci := range cartItems {
		ids = append(ids, ci.MenuItemID)
	}
	menuItems, err := loadMenuItems(ctx, itemTxRepo, ids)
	if err != nil {
		return nil, err
	}

	orderID := uuid.New().String()
	orderItems, totalAmount, err := buildOrderItems(cartItems, menuItems, orderID)
	if err != nil {
		return nil, err
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
		IdempotencyKey:  req.IdempotencyKey,
		Items:           orderItems,
	}

	if err := orderTxRepo.CreateWithItems(ctx, order, orderItems, paymentMethod); err != nil {
		// Параллельный запрос с тем же ключом мог успеть первым:
		// тогда отдаём его заказ вместо ошибки дубля.
		if req.IdempotencyKey != "" {
			if existing, getErr := s.orderRepo.GetByUserAndIdempotencyKey(ctx, userID, req.IdempotencyKey); getErr == nil {
				_ = tx.Rollback()
				return existing, nil
			}
		}
		return nil, fmt.Errorf("failed to create order: %w", err)
	}

	if err := cartTxRepo.Clear(ctx, cart.ID); err != nil {
		return nil, fmt.Errorf("failed to clear cart: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit order tx: %w", err)
	}

	return order, nil
}

// checkCreateRequest - проверяет входные данные заказа, возвращает способ оплаты.
func checkCreateRequest(req domain.CreateOrderRequest) (string, error) {
	if key := req.IdempotencyKey; key != "" {
		if len(key) > 255 || strings.TrimSpace(key) == "" {
			return "", fmt.Errorf("%w: idempotency key must contain 1 to 255 printable ASCII characters", domain.ErrValidation)
		}
		for i := 0; i < len(key); i++ {
			if key[i] < 0x21 || key[i] > 0x7e {
				return "", fmt.Errorf("%w: idempotency key must contain 1 to 255 printable ASCII characters", domain.ErrValidation)
			}
		}
	}
	if req.VenueID == "" {
		return "", fmt.Errorf("%w: venue_id is required", domain.ErrValidation)
	}
	if req.DeliveryType != string(domain.DeliveryTypeDelivery) && req.DeliveryType != string(domain.DeliveryTypePickup) {
		return "", fmt.Errorf("%w: invalid delivery type: %s", domain.ErrValidation, req.DeliveryType)
	}
	if req.DeliveryType == string(domain.DeliveryTypeDelivery) && strings.TrimSpace(req.DeliveryAddress) == "" {
		return "", fmt.Errorf("%w: delivery_address is required for delivery", domain.ErrValidation)
	}
	if req.PaymentMethod == "" {
		return defaultPaymentMethod, nil
	}
	if req.PaymentMethod != "card" && req.PaymentMethod != "cash" && req.PaymentMethod != "sbp" {
		return "", fmt.Errorf("%w: payment_method must be card, cash or sbp", domain.ErrValidation)
	}
	return req.PaymentMethod, nil
}

// findIdempotentOrder - ищет готовый заказ по ключу, done говорит, что дальше идти не надо.
func (s *OrderService) findIdempotentOrder(ctx context.Context, userID, key string) (*domain.Order, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	existing, err := s.orderRepo.GetByUserAndIdempotencyKey(ctx, userID, key)
	if err == nil {
		if existing.UserID != userID {
			return nil, true, domain.ErrForbidden
		}
		return existing, true, nil
	}
	if !errors.Is(err, domain.ErrNotFound) {
		return nil, true, err
	}
	return nil, false, nil
}

// loadCart - грузит корзину с позициями, пустая корзина даёт ErrValidation.
func (s *OrderService) loadCart(ctx context.Context, userID, venueID string) (*domain.Cart, []domain.CartItem, error) {
	cart, err := s.cartRepo.GetOrCreate(ctx, userID, venueID)
	if err != nil {
		return nil, nil, fmt.Errorf("cart not found: %w", err)
	}

	cartItems, err := s.cartRepo.GetItems(ctx, cart.ID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get cart items: %w", err)
	}

	if len(cartItems) == 0 {
		return nil, nil, fmt.Errorf("%w: cart is empty", domain.ErrValidation)
	}

	return cart, cartItems, nil
}

// buildOrderItems - проверяет наличие блюд и считает сумму позиций.
func buildOrderItems(cartItems []domain.CartItem, menuItems map[string]domain.MenuItem, orderID string) ([]domain.OrderItem, int64, error) {
	var totalAmount int64
	orderItems := make([]domain.OrderItem, 0, len(cartItems))

	for _, ci := range cartItems {
		menuItem, ok := menuItems[ci.MenuItemID]
		if !ok {
			return nil, 0, fmt.Errorf("menu item %s not found: %w", ci.MenuItemID, domain.ErrNotFound)
		}

		if !menuItem.IsAvailable {
			return nil, 0, fmt.Errorf("%w: menu item %s is not available", domain.ErrValidation, menuItem.Name)
		}

		lineAmount, err := checkedLineAmount(menuItem.Price, ci.Quantity)
		if err != nil {
			return nil, 0, err
		}
		if totalAmount > maxInt64-lineAmount {
			return nil, 0, fmt.Errorf("%w: order total exceeds the supported amount", domain.ErrValidation)
		}
		totalAmount += lineAmount

		orderItems = append(orderItems, domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    orderID,
			MenuItemID: ci.MenuItemID,
			Quantity:   ci.Quantity,
			Price:      menuItem.Price,
			Options:    normalizeOptions(ci.Options),
		})
	}

	return orderItems, totalAmount, nil
}

const maxInt64 = int64(1<<63 - 1)

func checkedLineAmount(price int64, quantity int) (int64, error) {
	if price < 0 || quantity <= 0 || quantity > domain.MaxCartItemQuantity {
		return 0, fmt.Errorf("%w: invalid price or quantity", domain.ErrValidation)
	}
	q := int64(quantity)
	if price > maxInt64/q {
		return 0, fmt.Errorf("%w: line amount exceeds the supported amount", domain.ErrValidation)
	}
	return price * q, nil
}

// GetByID - возвращает заказ по id.
func (s *OrderService) GetByID(ctx context.Context, id string) (*domain.Order, error) {
	return s.orderRepo.GetByID(ctx, id)
}

// GetByUser - возвращает детали заказа только его владельцу.
func (s *OrderService) GetByUser(ctx context.Context, userID, id string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByIDForUser(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	order.Items, err = s.orderRepo.GetItems(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	return order, nil
}

// GetByVenue - возвращает детали заказа только заведению-владельцу.
func (s *OrderService) GetByVenue(ctx context.Context, venueID, id string) (*domain.Order, error) {
	order, err := s.orderRepo.GetByIDForVenue(ctx, id, venueID)
	if err != nil {
		return nil, err
	}
	order.Items, err = s.orderRepo.GetItems(ctx, order.ID)
	if err != nil {
		return nil, fmt.Errorf("get order items: %w", err)
	}
	return order, nil
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

	return s.orderRepo.UpdateStatusFrom(ctx, id, order.Status, status)
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

	return s.orderRepo.UpdateStatusFrom(ctx, id, order.Status, status)
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

	return fmt.Errorf("%w: invalid status transition from %s to %s", domain.ErrValidation, from, to)
}

// loadMenuItems - грузит блюда одним запросом с локой строк и раскладывает по id.
func loadMenuItems(ctx context.Context, itemRepo domain.MenuItemRepository, ids []string) (map[string]domain.MenuItem, error) {
	items, err := itemRepo.GetByIDsForUpdate(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("failed to load menu items: %w", err)
	}

	byID := make(map[string]domain.MenuItem, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	return byID, nil
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

	tx, err := s.transactor.BeginTx(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin repeat tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	orderTxRepo := s.orderRepo.WithTx(tx)
	itemTxRepo := s.itemRepo.WithTx(tx)

	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.MenuItemID)
	}
	menuItems, err := loadMenuItems(ctx, itemTxRepo, ids)
	if err != nil {
		return nil, err
	}

	var totalAmount int64
	var newItems []domain.OrderItem
	newOrderID := uuid.New().String()

	for _, item := range items {
		menuItem, ok := menuItems[item.MenuItemID]
		if !ok {
			continue
		}

		if !menuItem.IsAvailable {
			continue
		}

		lineAmount, amountErr := checkedLineAmount(menuItem.Price, item.Quantity)
		if amountErr != nil || totalAmount > maxInt64-lineAmount {
			return nil, fmt.Errorf("%w: repeated order total exceeds the supported amount", domain.ErrValidation)
		}
		totalAmount += lineAmount

		newItems = append(newItems, domain.OrderItem{
			ID:         uuid.New().String(),
			OrderID:    newOrderID,
			MenuItemID: item.MenuItemID,
			Quantity:   item.Quantity,
			Price:      menuItem.Price,
			Options:    normalizeOptions(item.Options),
		})
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

	if err := orderTxRepo.CreateWithItems(ctx, order, newItems, defaultPaymentMethod); err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit repeat tx: %w", err)
	}

	return order, nil
}
