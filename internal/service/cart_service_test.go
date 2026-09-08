package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func cartTestSetup() (*stubCartRepo, *stubMenuItemRepo) {
	cart := &stubCartRepo{
		cart:  &domain.Cart{ID: "cart-1", UserID: "user-1", VenueID: "venue-1"},
		items: []domain.CartItem{},
	}
	items := &stubMenuItemRepo{items: map[string]*domain.MenuItem{
		"item-1": {ID: "item-1", VenueID: "venue-1", Name: "Кола", Price: 15000, IsAvailable: true},
	}}
	return cart, items
}

func TestCartAddItem(t *testing.T) {
	t.Run("новое блюдо добавляют", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)

		got, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: 2,
		})
		require.NoError(t, err)
		require.Len(t, cart.added, 1)
		require.Equal(t, 2, cart.added[0].Quantity)
		require.Equal(t, "cart-1", got.ID)
	})

	t.Run("повторное блюдо увеличивает количество", func(t *testing.T) {
		cart, items := cartTestSetup()
		cart.items = []domain.CartItem{{ID: "ci-1", CartID: "cart-1", MenuItemID: "item-1", Quantity: 1}}
		svc := NewCartService(cart, items)

		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: 2,
		})
		require.NoError(t, err)
		require.Empty(t, cart.added)
		require.Equal(t, 3, cart.updatedQty["ci-1"])
	})
}

func TestCartAddItemValidation(t *testing.T) {
	t.Run("нулевое количество", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: 0,
		})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, cart.added)
	})

	t.Run("отрицательное количество", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: -1,
		})
		require.ErrorIs(t, err, domain.ErrValidation)
	})

	t.Run("блюда нет в меню", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "nope", Quantity: 1,
		})
		require.Error(t, err)
		require.Empty(t, cart.added)
	})

	t.Run("блюдо снято с продажи", func(t *testing.T) {
		cart, items := cartTestSetup()
		items.items["item-1"].IsAvailable = false
		svc := NewCartService(cart, items)
		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: 1,
		})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, cart.added)
	})

	t.Run("блюдо чужого заведения", func(t *testing.T) {
		cart, items := cartTestSetup()
		items.items["item-1"].VenueID = "venue-2"
		svc := NewCartService(cart, items)
		_, err := svc.AddItem(context.Background(), "user-1", "venue-1", domain.AddToCartRequest{
			MenuItemID: "item-1", Quantity: 1,
		})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, cart.added)
	})
}

func TestCartManage(t *testing.T) {
	t.Run("смена количества уходит в репозиторий", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		require.NoError(t, svc.UpdateItemQuantity(context.Background(), "user-1", "ci-1", 5))
		require.Equal(t, 5, cart.updatedQty["ci-1"])
	})

	t.Run("позицию убирают", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		require.NoError(t, svc.RemoveItem(context.Background(), "user-1", "ci-1"))
		require.Equal(t, []string{"ci-1"}, cart.removed)
	})

	t.Run("корзину чистят целиком", func(t *testing.T) {
		cart, items := cartTestSetup()
		svc := NewCartService(cart, items)
		require.NoError(t, svc.Clear(context.Background(), "user-1", "venue-1"))
		require.Equal(t, []string{"cart-1"}, cart.cleared)
	})
}
