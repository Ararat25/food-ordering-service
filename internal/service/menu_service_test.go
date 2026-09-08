package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func menuTestSetup() *stubMenuItemRepo {
	return &stubMenuItemRepo{items: map[string]*domain.MenuItem{
		"item-1": {ID: "item-1", CategoryID: "cat-1", VenueID: "venue-1", Name: "Кола", Price: 15000, IsAvailable: true},
	}}
}

func TestMenuCreateItem(t *testing.T) {
	items := menuTestSetup()
	svc := NewMenuService(&stubMenuCategoryRepo{}, items)

	item := &domain.MenuItem{CategoryID: "cat-1", VenueID: "venue-1", Name: "Суп", Price: 25000}
	require.NoError(t, svc.CreateItem(context.Background(), item))
	require.NotEmpty(t, item.ID)
	require.Len(t, items.created, 1)
	// NULL в options раньше ронял чтение, в базу должны уходить [].
	require.JSONEq(t, `[]`, string(items.created[0].Options))
}

func TestMenuCreateItemValidation(t *testing.T) {
	t.Run("без имени не создают", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.CreateItem(context.Background(), &domain.MenuItem{VenueID: "venue-1", Price: 100})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, items.created)
	})

	t.Run("отрицательную цену не создают", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.CreateItem(context.Background(), &domain.MenuItem{VenueID: "venue-1", Name: "Суп", Price: -1})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, items.created)
	})

	t.Run("категорию без имени не создают", func(t *testing.T) {
		cats := &stubMenuCategoryRepo{}
		svc := NewMenuService(cats, menuTestSetup())
		err := svc.CreateCategory(context.Background(), &domain.MenuCategory{VenueID: "venue-1"})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, cats.created)
	})
}

func TestMenuVenueScoping(t *testing.T) {
	t.Run("обновить чужое блюдо", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.UpdateItem(context.Background(), &domain.MenuItem{ID: "item-1", VenueID: "venue-2", Name: "Хак"})
		require.ErrorIs(t, err, domain.ErrForbidden)
		require.Empty(t, items.updated)
	})

	t.Run("обновить своё блюдо", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.UpdateItem(context.Background(), &domain.MenuItem{ID: "item-1", VenueID: "venue-1", Name: "Кола 0.5"})
		require.NoError(t, err)
		require.Len(t, items.updated, 1)
	})

	t.Run("удалить чужое блюдо", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.DeleteItem(context.Background(), "venue-2", "item-1")
		require.ErrorIs(t, err, domain.ErrForbidden)
		require.Empty(t, items.deleted)
	})

	t.Run("удалить своё блюдо", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		require.NoError(t, svc.DeleteItem(context.Background(), "venue-1", "item-1"))
		require.Equal(t, []string{"item-1"}, items.deleted)
	})

	t.Run("наличие у чужого блюда не меняют", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.UpdateAvailability(context.Background(), "venue-2", "item-1", false)
		require.ErrorIs(t, err, domain.ErrForbidden)
		require.Empty(t, items.avail)
	})

	t.Run("блюда нет — 404, а не 403", func(t *testing.T) {
		items := menuTestSetup()
		svc := NewMenuService(&stubMenuCategoryRepo{}, items)
		err := svc.DeleteItem(context.Background(), "venue-2", "nope")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestMenuCreateCategory(t *testing.T) {
	cats := &stubMenuCategoryRepo{}
	svc := NewMenuService(cats, menuTestSetup())

	cat := &domain.MenuCategory{VenueID: "venue-1", Name: "Напитки"}
	require.NoError(t, svc.CreateCategory(context.Background(), cat))
	require.NotEmpty(t, cat.ID)
	require.Len(t, cats.created, 1)
}

func TestMenuListPagination(t *testing.T) {
	items := menuTestSetup()
	items.listTotal = 42
	cats := &stubMenuCategoryRepo{listTotal: 7}
	svc := NewMenuService(cats, items)

	gotItems, total, err := svc.ListItems(context.Background(), domain.MenuItemFilters{VenueID: "venue-1"})
	require.NoError(t, err)
	require.Equal(t, 42, total)
	require.Equal(t, domain.DefaultPage, items.listPage)
	require.Equal(t, domain.DefaultPerPage, items.listPer)
	require.Empty(t, gotItems)

	gotCats, total, err := svc.ListCategories(context.Background(), "venue-1", 2, 5)
	require.NoError(t, err)
	require.Equal(t, 7, total)
	require.Equal(t, 2, cats.listPage)
	require.Equal(t, 5, cats.listPer)
	require.Empty(t, gotCats)
}
