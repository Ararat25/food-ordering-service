package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// OrderHandler - обработчик для заказов, пользовательская часть и кабинет заведения.
type OrderHandler struct {
	orderService domain.OrderService // логика заказов
}

// NewOrderHandler - создаёт хендлер заказов.
func NewOrderHandler(orderService domain.OrderService) *OrderHandler {
	return &OrderHandler{orderService: orderService}
}

// Create - создаёт заказ из корзины пользователя.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	var req domain.CreateOrderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	order, err := h.orderService.Create(r.Context(), userID, req)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": order})
}

// List - отдаёт заказы пользователя постранично.
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	page, perPage := paginationFromQuery(r)

	orders, total, err := h.orderService.ListByUser(r.Context(), userID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, orders, page, perPage, total)
}

// GetByID - отдаёт заказ по id из пути.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, err := h.orderService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": order})
}

// Track - отдаёт короткий статус заказа для отслеживания.
func (h *OrderHandler) Track(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	order, err := h.orderService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"data": map[string]any{
			"id":        order.ID,
			"status":    order.Status,
			"estimated": order.EstimatedDeliveryTime,
		},
	})
}

// Repeat - повторяет заказ пользователя заново.
func (h *OrderHandler) Repeat(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	orderID := chi.URLParam(r, "id")

	order, err := h.orderService.Repeat(r.Context(), userID, orderID)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": order})
}

// VenueList - отдаёт заказы заведения из контекста с фильтром по статусу.
func (h *OrderHandler) VenueList(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())
	status := r.URL.Query().Get("status")
	page, perPage := paginationFromQuery(r)

	orders, total, err := h.orderService.ListByVenue(r.Context(), venueID, status, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, orders, page, perPage, total)
}

// VenueAccept - принимает заказ заведения.
func (h *OrderHandler) VenueAccept(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatusAccepted); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "order accepted"})
}

// VenueReject - отклоняет заказ заведения.
func (h *OrderHandler) VenueReject(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatusCancelled); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "order rejected"})
}

// VenueUpdateStatus - меняет статус заказа заведения на произвольный из тела.
func (h *OrderHandler) VenueUpdateStatus(w http.ResponseWriter, r *http.Request) {
	orderID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	var req struct {
		Status string `json:"status"` // новый статус
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatus(req.Status)); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "status updated"})
}
