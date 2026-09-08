package service

import (
	"context"
	"database/sql"
	"errors"

	"avito-kitchen/internal/domain"
)

var (
	_ domain.CartRepository         = (*stubCartRepo)(nil)
	_ domain.OrderRepository        = (*stubOrderRepo)(nil)
	_ domain.MenuItemRepository     = (*stubMenuItemRepo)(nil)
	_ domain.MenuCategoryRepository = (*stubMenuCategoryRepo)(nil)
	_ domain.VenueRepository        = (*stubVenueRepo)(nil)
	_ domain.ReviewRepository       = (*stubReviewRepo)(nil)
	_ domain.NotificationRepository = (*stubNotifRepo)(nil)
	_ domain.PushTokenRepository    = (*stubPushTokenRepo)(nil)
	_ domain.Transactor             = (*stubTransactor)(nil)
)

var errUnexpectedTx = errors.New("unexpected BeginTx call")

type stubTransactor struct {
	tx  domain.Tx
	err error
}

func (s *stubTransactor) BeginTx(_ context.Context) (domain.Tx, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.tx == nil {
		return nil, errUnexpectedTx
	}
	return s.tx, nil
}

type stubTx struct {
	committed  bool
	rolledBack bool
	commitErr  error
}

func (t *stubTx) ExecContext(_ context.Context, _ string, _ ...any) (sql.Result, error) {
	return nil, errors.New("stub tx: queries go through repositories")
}

func (t *stubTx) QueryContext(_ context.Context, _ string, _ ...any) (*sql.Rows, error) {
	return nil, errors.New("stub tx: queries go through repositories")
}

func (t *stubTx) QueryRowContext(_ context.Context, _ string, _ ...any) *sql.Row {
	return nil
}

func (t *stubTx) Commit() error {
	t.committed = true
	return t.commitErr
}

func (t *stubTx) Rollback() error {
	t.rolledBack = true
	return nil
}

type stubCartRepo struct {
	cart       *domain.Cart
	cartErr    error
	items      []domain.CartItem
	itemsErr   error
	addErr     error
	updateErr  error
	removeErr  error
	clearErr   error
	cleared    []string
	added      []*domain.CartItem
	updatedQty map[string]int
	removed    []string
}

func (s *stubCartRepo) WithTx(_ domain.Tx) domain.CartRepository { return s }

func (s *stubCartRepo) GetOrCreate(_ context.Context, _, _ string) (*domain.Cart, error) {
	if s.cartErr != nil {
		return nil, s.cartErr
	}
	return s.cart, nil
}

func (s *stubCartRepo) GetOrCreateForUpdate(_ context.Context, _, _ string) (*domain.Cart, error) {
	if s.cartErr != nil {
		return nil, s.cartErr
	}
	return s.cart, nil
}

func (s *stubCartRepo) GetByID(_ context.Context, _ string) (*domain.Cart, error) {
	return s.cart, nil
}

func (s *stubCartRepo) Clear(_ context.Context, cartID string) error {
	s.cleared = append(s.cleared, cartID)
	return s.clearErr
}

func (s *stubCartRepo) AddItem(_ context.Context, item *domain.CartItem) error {
	s.added = append(s.added, item)
	return s.addErr
}

func (s *stubCartRepo) UpdateItemQuantity(_ context.Context, itemID string, quantity int) error {
	if s.updatedQty == nil {
		s.updatedQty = map[string]int{}
	}
	s.updatedQty[itemID] = quantity
	return s.updateErr
}

func (s *stubCartRepo) RemoveItem(_ context.Context, itemID string) error {
	s.removed = append(s.removed, itemID)
	return s.removeErr
}

func (s *stubCartRepo) UpdateItemQuantityForUser(ctx context.Context, _ string, itemID string, quantity int) error {
	return s.UpdateItemQuantity(ctx, itemID, quantity)
}

func (s *stubCartRepo) RemoveItemForUser(ctx context.Context, _ string, itemID string) error {
	return s.RemoveItem(ctx, itemID)
}

func (s *stubCartRepo) GetItems(_ context.Context, _ string) ([]domain.CartItem, error) {
	return s.items, s.itemsErr
}

func (s *stubCartRepo) GetItemsForUpdate(ctx context.Context, cartID string) ([]domain.CartItem, error) {
	return s.GetItems(ctx, cartID)
}

type stubOrderRepo struct {
	order     *domain.Order
	orderErr  error
	items     []domain.OrderItem
	itemsErr  error
	createErr error
	statusErr error
	created   []domain.Order
	statuses  map[string]domain.OrderStatus
	payMethod string
	idemOrder *domain.Order
	idemErr   error
}

func (s *stubOrderRepo) WithTx(_ domain.Tx) domain.OrderRepository { return s }

func (s *stubOrderRepo) Create(_ context.Context, _ *domain.Order) error { return nil }

func (s *stubOrderRepo) CreateWithItems(_ context.Context, order *domain.Order, items []domain.OrderItem, payMethod string) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.payMethod = payMethod
	cp := *order
	cp.Items = items
	s.created = append(s.created, cp)
	return nil
}

func (s *stubOrderRepo) GetByIdempotencyKey(_ context.Context, _ string) (*domain.Order, error) {
	if s.idemErr != nil {
		return nil, s.idemErr
	}
	if s.idemOrder == nil {
		return nil, domain.ErrNotFound
	}
	return s.idemOrder, nil
}

func (s *stubOrderRepo) GetByUserAndIdempotencyKey(ctx context.Context, userID string, key string) (*domain.Order, error) {
	if s.idemOrder != nil && s.idemOrder.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return s.GetByIdempotencyKey(ctx, key)
}

func (s *stubOrderRepo) GetByID(_ context.Context, _ string) (*domain.Order, error) {
	if s.orderErr != nil {
		return nil, s.orderErr
	}
	return s.order, nil
}

func (s *stubOrderRepo) GetByIDForUser(_ context.Context, id, userID string) (*domain.Order, error) {
	order, err := s.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if order.UserID != userID {
		return nil, domain.ErrNotFound
	}
	return order, nil
}

func (s *stubOrderRepo) GetByIDForVenue(_ context.Context, id, venueID string) (*domain.Order, error) {
	order, err := s.GetByID(context.Background(), id)
	if err != nil {
		return nil, err
	}
	if order.VenueID != venueID {
		return nil, domain.ErrNotFound
	}
	return order, nil
}

func (s *stubOrderRepo) ListByUser(_ context.Context, _ string, _, _ int) ([]domain.Order, int, error) {
	return nil, 0, nil
}

func (s *stubOrderRepo) ListByVenue(_ context.Context, _ string, _ string, _, _ int) ([]domain.Order, int, error) {
	return nil, 0, nil
}

func (s *stubOrderRepo) UpdateStatusFrom(_ context.Context, id string, from, status domain.OrderStatus) error {
	if s.statusErr != nil {
		return s.statusErr
	}
	if s.order != nil && s.order.Status != from {
		return domain.ErrConflict
	}
	if s.statuses == nil {
		s.statuses = map[string]domain.OrderStatus{}
	}
	s.statuses[id] = status
	if s.order != nil {
		s.order.Status = status
	}
	return nil
}

func (s *stubOrderRepo) GetItems(_ context.Context, _ string) ([]domain.OrderItem, error) {
	return s.items, s.itemsErr
}

type stubMenuItemRepo struct {
	items      map[string]*domain.MenuItem
	getErr     error
	created    []*domain.MenuItem
	updated    []*domain.MenuItem
	deleted    []string
	avail      map[string]bool
	byIDsCalls int
	listPage   int
	listPer    int
	listTotal  int
}

func (s *stubMenuItemRepo) GetByID(_ context.Context, id string) (*domain.MenuItem, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	item, ok := s.items[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return item, nil
}

func (s *stubMenuItemRepo) Create(_ context.Context, item *domain.MenuItem) error {
	s.created = append(s.created, item)
	return nil
}

func (s *stubMenuItemRepo) Update(_ context.Context, item *domain.MenuItem) error {
	s.updated = append(s.updated, item)
	return nil
}

func (s *stubMenuItemRepo) Delete(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

func (s *stubMenuItemRepo) UpdateAvailability(_ context.Context, id string, available bool) error {
	if s.avail == nil {
		s.avail = map[string]bool{}
	}
	s.avail[id] = available
	return nil
}

func (s *stubMenuItemRepo) ListByVenue(_ context.Context, f domain.MenuItemFilters) ([]domain.MenuItem, int, error) {
	s.listPage, s.listPer = f.Page, f.PerPage
	return nil, s.listTotal, nil
}

func (s *stubMenuItemRepo) ListByCategory(_ context.Context, _ string, page, perPage int) ([]domain.MenuItem, int, error) {
	s.listPage, s.listPer = page, perPage
	return nil, s.listTotal, nil
}

func (s *stubMenuItemRepo) GetByIDs(_ context.Context, ids []string) ([]domain.MenuItem, error) {
	s.byIDsCalls++
	return s.findByIDs(ids)
}

func (s *stubMenuItemRepo) GetByIDsForUpdate(_ context.Context, ids []string) ([]domain.MenuItem, error) {
	s.byIDsCalls++
	return s.findByIDs(ids)
}

func (s *stubMenuItemRepo) findByIDs(ids []string) ([]domain.MenuItem, error) {
	var out []domain.MenuItem
	for _, id := range ids {
		item, ok := s.items[id]
		if !ok {
			continue
		}
		out = append(out, *item)
	}
	return out, s.getErr
}

func (s *stubMenuItemRepo) WithTx(_ domain.Tx) domain.MenuItemRepository { return s }

type stubMenuCategoryRepo struct {
	created   []*domain.MenuCategory
	updated   []*domain.MenuCategory
	deleted   []string
	listPage  int
	listPer   int
	listTotal int
}

func (s *stubMenuCategoryRepo) Create(_ context.Context, c *domain.MenuCategory) error {
	s.created = append(s.created, c)
	return nil
}

func (s *stubMenuCategoryRepo) GetByID(_ context.Context, _ string) (*domain.MenuCategory, error) {
	return nil, domain.ErrNotFound
}

func (s *stubMenuCategoryRepo) GetByIDForVenue(_ context.Context, _ string, _ string) (*domain.MenuCategory, error) {
	return &domain.MenuCategory{ID: "cat-1", VenueID: "venue-1"}, nil
}

func (s *stubMenuCategoryRepo) ListByVenue(_ context.Context, _ string, page, perPage int) ([]domain.MenuCategory, int, error) {
	s.listPage, s.listPer = page, perPage
	return nil, s.listTotal, nil
}

func (s *stubMenuCategoryRepo) Update(_ context.Context, c *domain.MenuCategory) error {
	s.updated = append(s.updated, c)
	return nil
}

func (s *stubMenuCategoryRepo) Delete(_ context.Context, id string) error {
	s.deleted = append(s.deleted, id)
	return nil
}

type stubVenueRepo struct {
	venue   *domain.Venue
	getErr  error
	keyErr  error
	gotList []domain.VenueFilters
	listOut []domain.Venue
	listN   int
	listErr error
	created []*domain.Venue
}

func (s *stubVenueRepo) Create(_ context.Context, v *domain.Venue) error {
	s.created = append(s.created, v)
	return nil
}

func (s *stubVenueRepo) GetByID(_ context.Context, _ string) (*domain.Venue, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return s.venue, nil
}

func (s *stubVenueRepo) GetByAPIKey(_ context.Context, _ string) (*domain.Venue, error) {
	if s.keyErr != nil {
		return nil, s.keyErr
	}
	return s.venue, nil
}

func (s *stubVenueRepo) List(_ context.Context, f domain.VenueFilters) ([]domain.Venue, int, error) {
	s.gotList = append(s.gotList, f)
	return s.listOut, s.listN, s.listErr
}

func (s *stubVenueRepo) Update(_ context.Context, _ *domain.Venue) error { return nil }

type stubReviewRepo struct {
	existing *domain.Review
	existErr error
	created  []*domain.Review
	gotList  [][2]int // принятые (page, perPage)
	listOut  []domain.Review
	listN    int
}

func (s *stubReviewRepo) Create(_ context.Context, r *domain.Review) error {
	s.created = append(s.created, r)
	return nil
}

func (s *stubReviewRepo) ListByVenue(_ context.Context, _ string, page, perPage int) ([]domain.Review, int, error) {
	s.gotList = append(s.gotList, [2]int{page, perPage})
	return s.listOut, s.listN, nil
}

func (s *stubReviewRepo) GetByUserAndOrder(_ context.Context, _, _ string) (*domain.Review, error) {
	if s.existErr != nil {
		return nil, s.existErr
	}
	return s.existing, nil
}

type stubNotifRepo struct {
	created []*domain.Notification
	read    []string
	gotList [][2]int
}

func (s *stubNotifRepo) Create(_ context.Context, n *domain.Notification) error {
	s.created = append(s.created, n)
	return nil
}

func (s *stubNotifRepo) ListByUser(_ context.Context, _ string, page, perPage int) ([]domain.Notification, int, error) {
	s.gotList = append(s.gotList, [2]int{page, perPage})
	return nil, 0, nil
}

func (s *stubNotifRepo) MarkAsRead(_ context.Context, id string) error {
	s.read = append(s.read, id)
	return nil
}

type stubPushTokenRepo struct {
	registered []*domain.PushToken
	revoked    []string
}

func (s *stubPushTokenRepo) Register(_ context.Context, t *domain.PushToken) error {
	s.registered = append(s.registered, t)
	return nil
}

func (s *stubPushTokenRepo) GetByUserID(_ context.Context, _ string) ([]domain.PushToken, error) {
	return nil, nil
}

func (s *stubPushTokenRepo) DeleteForUser(_ context.Context, userID, token string) error {
	s.revoked = append(s.revoked, userID+":"+token)
	return nil
}
