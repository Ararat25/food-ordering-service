package handler

import (
	"log/slog"
	"net/http"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// OrderHandler - обработчик для заказов, пользовательская часть и кабинет заведения.
type OrderHandler struct {
	orderService domain.OrderService // логика заказов
	logger       *slog.Logger        // куда писать внутренние ошибки
}

// NewOrderHandler - создаёт хендлер заказов.
func NewOrderHandler(orderService domain.OrderService, logger *slog.Logger) *OrderHandler {
	return &OrderHandler{orderService: orderService, logger: logger}
}

// Create - создаёт заказ из корзины пользователя.
func (h *OrderHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	var req domain.CreateOrderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	req.IdempotencyKey = r.Header.Get("Idempotency-Key")

	if !checkOptionalUUID(w, r, "venue_id", req.VenueID) {
		return
	}

	order, err := h.orderService.Create(r.Context(), userID, req)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": order})
}

// List - отдаёт заказы пользователя постранично.
func (h *OrderHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	page, perPage, err := paginationFromQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	orders, total, err := h.orderService.ListByUser(r.Context(), userID, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, orders, page, perPage, total)
}

// GetByID - отдаёт заказ по id из пути.
func (h *OrderHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}

	order, err := h.orderService.GetByUser(r.Context(), userID, id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": order})
}

// VenueGetByID - отдаёт детали заказа только заведению-владельцу.
func (h *OrderHandler) VenueGetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}
	venueID, _ := middleware.GetVenueID(r.Context())
	order, err := h.orderService.GetByVenue(r.Context(), venueID, id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"data": order})
}

// Track - отдаёт короткий статус заказа для отслеживания.
func (h *OrderHandler) Track(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}

	order, err := h.orderService.GetByUser(r.Context(), userID, id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
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
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	orderID, ok := idFromPath(w, r)
	if !ok {
		return
	}

	order, err := h.orderService.Repeat(r.Context(), userID, orderID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": order})
}

// VenueList - отдаёт заказы заведения из контекста с фильтром по статусу.
func (h *OrderHandler) VenueList(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())
	status := r.URL.Query().Get("status")
	if status != "" {
		switch domain.OrderStatus(status) {
		case domain.OrderStatusPending, domain.OrderStatusAccepted, domain.OrderStatusPreparing,
			domain.OrderStatusReady, domain.OrderStatusDelivering, domain.OrderStatusDelivered,
			domain.OrderStatusCancelled:
		default:
			writeError(w, r, http.StatusBadRequest, "invalid status filter")
			return
		}
	}
	page, perPage, err := paginationFromQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	orders, total, err := h.orderService.ListByVenue(r.Context(), venueID, status, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, orders, page, perPage, total)
}

// VenueAccept - принимает заказ заведения.
func (h *OrderHandler) VenueAccept(w http.ResponseWriter, r *http.Request) {
	orderID, ok := idFromPath(w, r)
	if !ok {
		return
	}
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatusAccepted); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "order accepted"})
}

// VenueReject - отклоняет заказ заведения.
func (h *OrderHandler) VenueReject(w http.ResponseWriter, r *http.Request) {
	orderID, ok := idFromPath(w, r)
	if !ok {
		return
	}
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatusCancelled); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "order rejected"})
}

// VenueUpdateStatus - меняет статус заказа заведения на произвольный из тела.
func (h *OrderHandler) VenueUpdateStatus(w http.ResponseWriter, r *http.Request) {
	orderID, ok := idFromPath(w, r)
	if !ok {
		return
	}
	venueID, _ := middleware.GetVenueID(r.Context())

	var req struct {
		Status string `json:"status"` // новый статус
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}

	if err := h.orderService.UpdateVenueOrderStatus(r.Context(), venueID, orderID, domain.OrderStatus(req.Status)); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "status updated"})
}
