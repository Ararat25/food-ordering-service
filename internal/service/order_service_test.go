package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
	"avito-kitchen/internal/repository"
)

func orderTestCart() (*stubCartRepo, *stubMenuItemRepo) {
	cart := &stubCartRepo{
		cart:  &domain.Cart{ID: "cart-1", UserID: "user-1", VenueID: "venue-1"},
		items: []domain.CartItem{{ID: "ci-1", CartID: "cart-1", MenuItemID: "item-1", Quantity: 2}},
	}
	items := &stubMenuItemRepo{items: map[string]*domain.MenuItem{
		"item-1": {ID: "item-1", VenueID: "venue-1", Name: "Кола", Price: 15000, IsAvailable: true},
	}}
	return cart, items
}

func TestOrderCreateValidation(t *testing.T) {
	cart, items := orderTestCart()

	tests := []struct {
		name string
		req  domain.CreateOrderRequest
	}{
		{"пустое заведение", domain.CreateOrderRequest{DeliveryType: "delivery"}},
		{"левый тип доставки", domain.CreateOrderRequest{VenueID: "venue-1", DeliveryType: "drone"}},
		{"пустой тип доставки", domain.CreateOrderRequest{VenueID: "venue-1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewOrderService(&stubOrderRepo{}, cart, items, &stubTransactor{})
			_, err := svc.Create(context.Background(), "user-1", tt.req)
			require.ErrorIs(t, err, domain.ErrValidation)
		})
	}
}

func TestOrderCreateEmptyCart(t *testing.T) {
	cart, items := orderTestCart()
	cart.items = nil

	svc := NewOrderService(&stubOrderRepo{}, cart, items, &stubTransactor{tx: &stubTx{}})
	_, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
		VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1",
	})
	require.ErrorIs(t, err, domain.ErrValidation)
	require.Contains(t, err.Error(), "cart is empty")
}

func TestOrderCreateItemProblems(t *testing.T) {
	t.Run("блюда нет в меню", func(t *testing.T) {
		cart, items := orderTestCart()
		items.items = map[string]*domain.MenuItem{}

		svc := NewOrderService(&stubOrderRepo{}, cart, items, &stubTransactor{tx: &stubTx{}})
		_, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "pickup",
		})
		require.Error(t, err)
	})

	t.Run("блюдо снято с продажи", func(t *testing.T) {
		cart, items := orderTestCart()
		items.items["item-1"].IsAvailable = false

		svc := NewOrderService(&stubOrderRepo{}, cart, items, &stubTransactor{tx: &stubTx{}})
		_, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "pickup",
		})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Contains(t, err.Error(), "not available")
	})
}

func TestOrderCreateLoadsItemsInOneBatch(t *testing.T) {
	orderRepo := &stubOrderRepo{
		order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusDelivered},
		items: []domain.OrderItem{
			{ID: "oi-1", OrderID: "o-1", MenuItemID: "item-1", Quantity: 2},
			{ID: "oi-2", OrderID: "o-1", MenuItemID: "item-2", Quantity: 1},
		},
	}
	items := &stubMenuItemRepo{items: map[string]*domain.MenuItem{
		"item-1": {ID: "item-1", VenueID: "v-1", Price: 15000, IsAvailable: true},
		"item-2": {ID: "item-2", VenueID: "v-1", Price: 25000, IsAvailable: true},
	}}
	svc := NewOrderService(orderRepo, &stubCartRepo{}, items, &stubTransactor{tx: &stubTx{}})

	order, err := svc.Repeat(context.Background(), "u-1", "o-1")
	require.NoError(t, err)
	require.Equal(t, 1, items.byIDsCalls)
	require.Equal(t, int64(55000), order.TotalAmount)
}

// Проверяем границы транзакции через настоящие репозитории поверх sqlmock
func TestOrderCreateCommitsAll(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	_, items := orderTestCart()
	svc := NewOrderService(
		repository.NewOrderRepo(mockDB),
		repository.NewCartRepo(mockDB),
		items,
		repository.NewTransactor(mockDB),
	)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, venue_id, created_at, updated_at FROM carts").WithArgs("user-1", "venue-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "venue_id", "created_at", "updated_at"}).
			AddRow("cart-1", "user-1", "venue-1", time.Now(), time.Now()),
	)
	mock.ExpectQuery("SELECT id, cart_id, menu_item_id, quantity, options, created_at FROM cart_items").WithArgs("cart-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "cart_id", "menu_item_id", "quantity", "options", "created_at"}).
			AddRow("ci-1", "cart-1", "item-1", 2, []byte("[]"), time.Now()),
	)
	mock.ExpectExec("INSERT INTO orders").WithArgs(
		sqlmock.AnyArg(), "user-1", "venue-1", domain.OrderStatusPending,
		domain.DeliveryType("delivery"), sqlmock.AnyArg(), sqlmock.AnyArg(),
		sqlmock.AnyArg(), "", int64(30000), nil,
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO order_items").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), "item-1", 2, int64(15000), sqlmock.AnyArg(),
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO payments").WithArgs(
		sqlmock.AnyArg(), sqlmock.AnyArg(), domain.PaymentStatusPending, int64(30000), "card",
	).WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM cart_items").WithArgs("cart-1").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	order, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
		VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1",
	})
	require.NoError(t, err)
	require.Equal(t, "user-1", order.UserID)
	require.Equal(t, "venue-1", order.VenueID)
	require.Equal(t, domain.OrderStatusPending, order.Status)
	require.Equal(t, int64(30000), order.TotalAmount)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderCreateRollsBack(t *testing.T) {
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer func() { _ = mockDB.Close() }()

	_, items := orderTestCart()
	svc := NewOrderService(
		repository.NewOrderRepo(mockDB),
		repository.NewCartRepo(mockDB),
		items,
		repository.NewTransactor(mockDB),
	)

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT id, user_id, venue_id, created_at, updated_at FROM carts").WithArgs("user-1", "venue-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "user_id", "venue_id", "created_at", "updated_at"}).
			AddRow("cart-1", "user-1", "venue-1", time.Now(), time.Now()),
	)
	mock.ExpectQuery("SELECT id, cart_id, menu_item_id, quantity, options, created_at FROM cart_items").WithArgs("cart-1").WillReturnRows(
		sqlmock.NewRows([]string{"id", "cart_id", "menu_item_id", "quantity", "options", "created_at"}).
			AddRow("ci-1", "cart-1", "item-1", 2, []byte("[]"), time.Now()),
	)
	mock.ExpectExec("INSERT INTO orders").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO order_items").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO payments").WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("DELETE FROM cart_items").WillReturnError(errors.New("connection lost"))
	mock.ExpectRollback()

	_, err = svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
		VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1",
	})
	require.ErrorContains(t, err, "failed to clear cart")
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestOrderStatusTransitions(t *testing.T) {
	newSvc := func(status domain.OrderStatus) (*OrderService, *stubOrderRepo) {
		repo := &stubOrderRepo{order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: status}}
		return NewOrderService(repo, &stubCartRepo{}, &stubMenuItemRepo{}, &stubTransactor{}), repo
	}

	t.Run("pending принимают", func(t *testing.T) {
		svc, repo := newSvc(domain.OrderStatusPending)
		require.NoError(t, svc.UpdateStatus(context.Background(), "o-1", domain.OrderStatusAccepted))
		require.Equal(t, domain.OrderStatusAccepted, repo.statuses["o-1"])
	})

	t.Run("pending сразу в delivered нельзя", func(t *testing.T) {
		svc, repo := newSvc(domain.OrderStatusPending)
		err := svc.UpdateStatus(context.Background(), "o-1", domain.OrderStatusDelivered)
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, repo.statuses)
	})

	t.Run("вся цепочка доставки", func(t *testing.T) {
		for _, step := range [][2]domain.OrderStatus{
			{domain.OrderStatusAccepted, domain.OrderStatusPreparing},
			{domain.OrderStatusPreparing, domain.OrderStatusReady},
			{domain.OrderStatusReady, domain.OrderStatusDelivering},
			{domain.OrderStatusDelivering, domain.OrderStatusDelivered},
		} {
			svc, repo := newSvc(step[0])
			require.NoError(t, svc.UpdateStatus(context.Background(), "o-1", step[1]))
			require.Equal(t, step[1], repo.statuses["o-1"])
		}
	})

	t.Run("заказ не найден", func(t *testing.T) {
		repo := &stubOrderRepo{orderErr: domain.ErrNotFound}
		svc := NewOrderService(repo, &stubCartRepo{}, &stubMenuItemRepo{}, &stubTransactor{})
		err := svc.UpdateStatus(context.Background(), "o-1", domain.OrderStatusAccepted)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestOrderVenueScoping(t *testing.T) {
	repo := &stubOrderRepo{order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusPending}}
	svc := NewOrderService(repo, &stubCartRepo{}, &stubMenuItemRepo{}, &stubTransactor{})

	t.Run("чужое заведение", func(t *testing.T) {
		err := svc.UpdateVenueOrderStatus(context.Background(), "v-2", "o-1", domain.OrderStatusAccepted)
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("своё заведение", func(t *testing.T) {
		err := svc.UpdateVenueOrderStatus(context.Background(), "v-1", "o-1", domain.OrderStatusAccepted)
		require.NoError(t, err)
		require.Equal(t, domain.OrderStatusAccepted, repo.statuses["o-1"])
	})
}

func TestOrderRepeat(t *testing.T) {
	t.Run("повтор доступен и считает сумму заново", func(t *testing.T) {
		orderRepo := &stubOrderRepo{
			order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusDelivered},
			items: []domain.OrderItem{{ID: "oi-1", OrderID: "o-1", MenuItemID: "item-1", Quantity: 2}},
		}
		items := &stubMenuItemRepo{items: map[string]*domain.MenuItem{
			"item-1": {ID: "item-1", VenueID: "v-1", Price: 15000, IsAvailable: true},
		}}
		svc := NewOrderService(orderRepo, &stubCartRepo{}, items, &stubTransactor{tx: &stubTx{}})

		order, err := svc.Repeat(context.Background(), "u-1", "o-1")
		require.NoError(t, err)
		require.Equal(t, domain.OrderStatusPending, order.Status)
		require.Equal(t, int64(30000), order.TotalAmount)
		require.NotEqual(t, "o-1", order.ID)
		require.Len(t, orderRepo.created, 1)
	})

	t.Run("чужой заказ повторять нельзя", func(t *testing.T) {
		orderRepo := &stubOrderRepo{
			order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusDelivered},
		}
		svc := NewOrderService(orderRepo, &stubCartRepo{}, &stubMenuItemRepo{}, &stubTransactor{})
		_, err := svc.Repeat(context.Background(), "u-2", "o-1")
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("всё снято с продажи — повторять нечего", func(t *testing.T) {
		orderRepo := &stubOrderRepo{
			order: &domain.Order{ID: "o-1", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusDelivered},
			items: []domain.OrderItem{{ID: "oi-1", OrderID: "o-1", MenuItemID: "item-1", Quantity: 1}},
		}
		items := &stubMenuItemRepo{items: map[string]*domain.MenuItem{
			"item-1": {ID: "item-1", VenueID: "v-1", Price: 15000, IsAvailable: false},
		}}
		svc := NewOrderService(orderRepo, &stubCartRepo{}, items, &stubTransactor{tx: &stubTx{}})
		_, err := svc.Repeat(context.Background(), "u-1", "o-1")
		require.Error(t, err)
		require.Empty(t, orderRepo.created)
	})
}

func TestOrderCreateIdempotency(t *testing.T) {
	t.Run("повтор с тем же ключом возвращает готовый заказ", func(t *testing.T) {
		existing := &domain.Order{ID: "o-old", UserID: "u-1", VenueID: "v-1", Status: domain.OrderStatusPending}
		orderRepo := &stubOrderRepo{idemOrder: existing}
		cart, items := orderTestCart()
		svc := NewOrderService(orderRepo, cart, items, &stubTransactor{tx: &stubTx{}})

		got, err := svc.Create(context.Background(), "u-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1", IdempotencyKey: "key-1",
		})
		require.NoError(t, err)
		require.Equal(t, "o-old", got.ID)
		require.Empty(t, orderRepo.created)
	})

	t.Run("тот же ключ у другого пользователя не конфликтует", func(t *testing.T) {
		existing := &domain.Order{ID: "o-old", UserID: "u-2", VenueID: "v-1", Status: domain.OrderStatusPending}
		orderRepo := &stubOrderRepo{idemOrder: existing}
		cart, items := orderTestCart()
		svc := NewOrderService(orderRepo, cart, items, &stubTransactor{tx: &stubTx{}})

		got, err := svc.Create(context.Background(), "u-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1", IdempotencyKey: "key-1",
		})
		require.NoError(t, err)
		require.NotNil(t, got)
		require.Equal(t, "u-1", got.UserID)
		require.Len(t, orderRepo.created, 1)
	})
}

func TestOrderCreatePaymentMethod(t *testing.T) {
	t.Run("метод из запроса уходит в платёж", func(t *testing.T) {
		orderRepo := &stubOrderRepo{}
		cart, items := orderTestCart()
		svc := NewOrderService(orderRepo, cart, items, &stubTransactor{tx: &stubTx{}})

		_, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1", PaymentMethod: "cash",
		})
		require.NoError(t, err)
		require.Equal(t, "cash", orderRepo.payMethod)
	})

	t.Run("без метода подставляют карту", func(t *testing.T) {
		orderRepo := &stubOrderRepo{}
		cart, items := orderTestCart()
		svc := NewOrderService(orderRepo, cart, items, &stubTransactor{tx: &stubTx{}})

		_, err := svc.Create(context.Background(), "user-1", domain.CreateOrderRequest{
			VenueID: "venue-1", DeliveryType: "delivery", DeliveryAddress: "ул. Тестовая, 1",
		})
		require.NoError(t, err)
		require.Equal(t, "card", orderRepo.payMethod)
	})
}

func TestBuildOrderItemsRejectsInt64Overflow(t *testing.T) {
	t.Run("multiplication overflow", func(t *testing.T) {
		items, total, err := buildOrderItems(
			[]domain.CartItem{{MenuItemID: "item-1", Quantity: 2}},
			map[string]domain.MenuItem{"item-1": {ID: "item-1", Price: maxInt64, IsAvailable: true}},
			"order-1",
		)
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Nil(t, items)
		require.Zero(t, total)
	})

	t.Run("sum overflow", func(t *testing.T) {
		price := maxInt64/2 + 1
		items, total, err := buildOrderItems(
			[]domain.CartItem{{MenuItemID: "item-1", Quantity: 1}, {MenuItemID: "item-2", Quantity: 1}},
			map[string]domain.MenuItem{
				"item-1": {ID: "item-1", Price: price, IsAvailable: true},
				"item-2": {ID: "item-2", Price: price, IsAvailable: true},
			},
			"order-1",
		)
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Nil(t, items)
		require.Zero(t, total)
	})

	t.Run("max int64 without overflow is accepted", func(t *testing.T) {
		items, total, err := buildOrderItems(
			[]domain.CartItem{{MenuItemID: "item-1", Quantity: 1}},
			map[string]domain.MenuItem{"item-1": {ID: "item-1", Price: maxInt64, IsAvailable: true}},
			"order-1",
		)
		require.NoError(t, err)
		require.Len(t, items, 1)
		require.Equal(t, maxInt64, total)
	})
}

func TestCheckCreateRequestIdempotencyKeyFormat(t *testing.T) {
	base := domain.CreateOrderRequest{VenueID: "v-1", DeliveryType: "delivery", DeliveryAddress: "address"}
	for _, key := range []string{"", "valid-key_123", strings.Repeat("x", 256), "with space", "ключ"} {
		req := base
		req.IdempotencyKey = key
		_, err := checkCreateRequest(req)
		if key == "" || key == "valid-key_123" {
			require.NoError(t, err)
		} else {
			require.ErrorIs(t, err, domain.ErrValidation)
		}
	}
}
