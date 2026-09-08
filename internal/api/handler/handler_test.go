package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

var (
	_ domain.OrderService = (*stubOrderService)(nil)
	_ domain.MenuService  = (*stubMenuService)(nil)
	_ domain.VenueService = (*stubVenueService)(nil)
	_ domain.CartService  = (*stubCartService)(nil)
)

// Тестовые id обязаны быть настоящими UUID: хендлеры проверяют их на границе API.
const (
	testVenueID  = "3f6c0f34-0000-4000-8000-000000000001"
	testOrderID  = "3f6c0f34-0000-4000-8000-000000000002"
	testItemID   = "3f6c0f34-0000-4000-8000-000000000003"
	brokenItemID = "nope"
)

type stubOrderService struct {
	order   *domain.Order
	err     error
	gotReq  domain.CreateOrderRequest
	gotUser string
}

func (s *stubOrderService) Create(_ context.Context, userID string, req domain.CreateOrderRequest) (*domain.Order, error) {
	s.gotUser, s.gotReq = userID, req
	if s.err != nil {
		return nil, s.err
	}
	return s.order, nil
}

func (s *stubOrderService) GetByID(_ context.Context, _ string) (*domain.Order, error) {
	return s.order, s.err
}

func (s *stubOrderService) GetByUser(_ context.Context, _, _ string) (*domain.Order, error) {
	return s.order, s.err
}

func (s *stubOrderService) GetByVenue(_ context.Context, _, _ string) (*domain.Order, error) {
	return s.order, s.err
}

func (s *stubOrderService) ListByUser(_ context.Context, _ string, _, _ int) ([]domain.Order, int, error) {
	return nil, 0, s.err
}

func (s *stubOrderService) ListByVenue(_ context.Context, _ string, _ string, _, _ int) ([]domain.Order, int, error) {
	return nil, 0, s.err
}

func (s *stubOrderService) UpdateStatus(_ context.Context, _ string, _ domain.OrderStatus) error {
	return s.err
}

func (s *stubOrderService) UpdateVenueOrderStatus(_ context.Context, _ string, _ string, _ domain.OrderStatus) error {
	return s.err
}

func (s *stubOrderService) GetItems(_ context.Context, _ string) ([]domain.OrderItem, error) {
	return nil, s.err
}

func (s *stubOrderService) Repeat(_ context.Context, _ string, _ string) (*domain.Order, error) {
	return s.order, s.err
}

type stubMenuService struct {
	items []domain.MenuItem
	total int
	err   error
	got   domain.MenuItemFilters
}

func (s *stubMenuService) ListItems(_ context.Context, f domain.MenuItemFilters) ([]domain.MenuItem, int, error) {
	s.got = f
	return s.items, s.total, s.err
}

func (s *stubMenuService) ListCategories(_ context.Context, _ string, _, _ int) ([]domain.MenuCategory, int, error) {
	return nil, 0, s.err
}

func (s *stubMenuService) CreateCategory(_ context.Context, _ *domain.MenuCategory) error {
	return s.err
}

func (s *stubMenuService) UpdateCategory(_ context.Context, _ *domain.MenuCategory) error {
	return s.err
}

func (s *stubMenuService) DeleteCategory(_ context.Context, _ string) error { return s.err }

func (s *stubMenuService) GetItem(_ context.Context, _ string) (*domain.MenuItem, error) {
	return nil, s.err
}

func (s *stubMenuService) CreateItem(_ context.Context, _ *domain.MenuItem) error {
	return s.err
}

func (s *stubMenuService) UpdateItem(_ context.Context, _ *domain.MenuItem) error {
	return s.err
}

func (s *stubMenuService) DeleteItem(_ context.Context, _, _ string) error { return s.err }

func (s *stubMenuService) UpdateAvailability(_ context.Context, _, _ string, _ bool) error {
	return s.err
}

type stubVenueService struct {
	venue *domain.Venue
	err   error
}

func (s *stubVenueService) List(_ context.Context, _ domain.VenueFilters) ([]domain.Venue, int, error) {
	return nil, 0, s.err
}

func (s *stubVenueService) GetByID(_ context.Context, _ string) (*domain.Venue, error) {
	return s.venue, s.err
}

func (s *stubVenueService) GetByAPIKey(_ context.Context, _ string) (*domain.Venue, error) {
	return s.venue, s.err
}

func (s *stubVenueService) Create(_ context.Context, _ *domain.Venue) error { return s.err }

type stubCartService struct {
	cart *domain.Cart
	err  error
}

func (s *stubCartService) GetCart(_ context.Context, _, _ string) (*domain.Cart, error) {
	return s.cart, s.err
}

func (s *stubCartService) AddItem(_ context.Context, _, _ string, _ domain.AddToCartRequest) (*domain.Cart, error) {
	return s.cart, s.err
}

func (s *stubCartService) UpdateItemQuantity(_ context.Context, _ string, _ string, _ int) error {
	return s.err
}

func (s *stubCartService) RemoveItem(_ context.Context, _ string, _ string) error {
	return s.err
}

func (s *stubCartService) Clear(_ context.Context, _ string, _ string) error { return s.err }

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func withRequestID(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), middleware.RequestIDKey, "req-1"))
}

func TestOrderCreateHandler(t *testing.T) {
	body := `{"venue_id":"` + testVenueID + `","delivery_type":"delivery"}`

	t.Run("успех и ключ идемпотентности доходит до сервиса", func(t *testing.T) {
		svc := &stubOrderService{order: &domain.Order{ID: "o-1"}}
		h := NewOrderHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		r.Header.Set("X-User-ID", "u-1")
		r.Header.Set("Idempotency-Key", "key-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusCreated, w.Code)
		require.Equal(t, "u-1", svc.gotUser)
		require.Equal(t, "key-1", svc.gotReq.IdempotencyKey)
		require.Contains(t, w.Body.String(), "o-1")
	})

	t.Run("без пользователя 401", func(t *testing.T) {
		h := NewOrderHandler(&stubOrderService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("битый JSON 400", func(t *testing.T) {
		h := NewOrderHandler(&stubOrderService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(`{oops`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("лишнее поле 400", func(t *testing.T) {
		h := NewOrderHandler(&stubOrderService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(`{"venue_id":"`+testVenueID+`","nonsense":1}`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("битый venue_id 400", func(t *testing.T) {
		h := NewOrderHandler(&stubOrderService{}, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(`{"venue_id":"v-1","delivery_type":"delivery"}`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "venue_id must be a UUID")
	})

	t.Run("ошибка валидации 400 с текстом", func(t *testing.T) {
		svc := &stubOrderService{err: fmt.Errorf("%w: cart is empty", domain.ErrValidation)}
		h := NewOrderHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "cart is empty")
	})

	t.Run("внутренняя ошибка прячется, request_id на месте", func(t *testing.T) {
		svc := &stubOrderService{err: errors.New("pq: connection reset by peer")}
		h := NewOrderHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/orders", strings.NewReader(body))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.Create(w, withRequestID(r))

		require.Equal(t, http.StatusInternalServerError, w.Code)
		require.NotContains(t, w.Body.String(), "pq:")
		require.Contains(t, w.Body.String(), "internal error")
		require.Contains(t, w.Body.String(), "req-1")
	})
}

func TestMenuListHandler(t *testing.T) {
	t.Run("пагинация из query доходит до сервиса", func(t *testing.T) {
		svc := &stubMenuService{items: []domain.MenuItem{{ID: "i-1"}}, total: 5}
		h := NewMenuHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/venue/menu?page=2&per_page=1", nil)
		r = withRequestID(r)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", testVenueID)
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.ListByVenuePublic(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, testVenueID, svc.got.VenueID)
		require.Equal(t, 2, svc.got.Page)
		require.Equal(t, 1, svc.got.PerPage)
		require.Contains(t, w.Body.String(), `"total":5`)
	})

	t.Run("фильтры поиска и наличия доходят до сервиса", func(t *testing.T) {
		svc := &stubMenuService{}
		h := NewMenuHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/venues/"+testVenueID+"/menu?search=кола&available=true", nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", testVenueID)
		r = r.WithContext(context.WithValue(withRequestID(r).Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.ListByVenuePublic(w, r)

		require.Equal(t, http.StatusOK, w.Code)
		require.Equal(t, "кола", svc.got.Search)
		require.NotNil(t, svc.got.Available)
		require.True(t, *svc.got.Available)
	})

	t.Run("кривая пагинация отклоняется", func(t *testing.T) {
		svc := &stubMenuService{}
		h := NewMenuHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/venue/menu?page=abc", nil)
		w := httptest.NewRecorder()

		h.ListItems(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Zero(t, svc.got.Page)
	})
}

func TestVenueGetHandler(t *testing.T) {
	t.Run("нет заведения 404", func(t *testing.T) {
		svc := &stubVenueService{err: domain.ErrNotFound}
		h := NewVenueHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/venues/"+testVenueID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", testVenueID)
		r = r.WithContext(context.WithValue(withRequestID(r).Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.GetByID(w, r)

		require.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("кривой id в пути 400", func(t *testing.T) {
		svc := &stubVenueService{venue: &domain.Venue{ID: testVenueID}}
		h := NewVenueHandler(svc, testLogger())

		r := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/api/v1/venues/"+brokenItemID, nil)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("id", brokenItemID)
		r = r.WithContext(context.WithValue(withRequestID(r).Context(), chi.RouteCtxKey, rctx))
		w := httptest.NewRecorder()

		h.GetByID(w, r)

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "id must be a UUID")
	})
}

func TestCartAddHandler(t *testing.T) {
	t.Run("без venue_id 400", func(t *testing.T) {
		h := NewCartHandler(&stubCartService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/cart/items", strings.NewReader(`{"menu_item_id":"`+testItemID+`"}`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.AddItem(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("кривой venue_id в query 400", func(t *testing.T) {
		h := NewCartHandler(&stubCartService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/cart/items?venue_id=v-1", strings.NewReader(`{"menu_item_id":"`+testItemID+`"}`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.AddItem(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "venue_id must be a UUID")
	})

	t.Run("кривой menu_item_id в теле 400", func(t *testing.T) {
		h := NewCartHandler(&stubCartService{}, testLogger())
		r := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/api/v1/cart/items?venue_id="+testVenueID, strings.NewReader(`{"menu_item_id":"`+brokenItemID+`"}`))
		r.Header.Set("X-User-ID", "u-1")
		w := httptest.NewRecorder()

		h.AddItem(w, withRequestID(r))

		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Contains(t, w.Body.String(), "menu_item_id must be a UUID")
	})
}

func TestDecodeJSONRejectsTrailingValueAndOversizedBody(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Value string `json:"value"`
		}
		if err := decodeJSON(w, r, &payload); err != nil {
			writeDecodeError(w, r, err)
			return
		}
		writeJSON(w, http.StatusOK, payload)
	})

	t.Run("trailing JSON value", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{"value":"one"} {"value":"two"}`))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, withRequestID(r))
		require.Equal(t, http.StatusBadRequest, w.Code)
		require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})

	t.Run("body over 1 MiB", func(t *testing.T) {
		body := `{"value":"` + strings.Repeat("x", maxBodyBytes) + `"}`
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, withRequestID(r))
		require.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
		require.Equal(t, "application/json", w.Header().Get("Content-Type"))
	})
}

func TestMenuQueryRejectsInvalidAvailable(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/api/v1/venues/venue-1/menu?available=notabool", nil)
	_, err := menuFiltersFromQuery(r, "venue-1")
	require.ErrorIs(t, err, domain.ErrValidation)
}

func TestPaginationFromQueryRejectsInvalidAndOutOfRangeValues(t *testing.T) {
	for _, target := range []string{
		"/?page=abc",
		"/?page=0",
		"/?page=999999999999999999999999",
		"/?per_page=0",
		"/?per_page=101",
	} {
		r := httptest.NewRequest(http.MethodGet, target, nil)
		_, _, err := paginationFromQuery(r)
		require.ErrorIs(t, err, domain.ErrValidation, "target: %s", target)
	}
	page, perPage, err := paginationFromQuery(httptest.NewRequest(http.MethodGet, "/", nil))
	require.NoError(t, err)
	require.Equal(t, domain.DefaultPage, page)
	require.Equal(t, domain.DefaultPerPage, perPage)
}
