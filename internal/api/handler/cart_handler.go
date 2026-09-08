package handler

import (
	"log/slog"
	"net/http"

	"avito-kitchen/internal/domain"
)

// CartHandler - обработчик для корзины.
type CartHandler struct {
	cartService domain.CartService // логика корзины
	logger      *slog.Logger       // куда писать внутренние ошибки
}

// NewCartHandler - создаёт хендлер корзины.
func NewCartHandler(cartService domain.CartService, logger *slog.Logger) *CartHandler {
	return &CartHandler{cartService: cartService, logger: logger}
}

// Get - отдаёт корзину пользователя в заведении из query.
func (h *CartHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID, ok := venueIDFromQuery(w, r)
	if !ok {
		return
	}

	cart, err := h.cartService.GetCart(r.Context(), userID, venueID)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

// AddItem - кладёт блюдо в корзину, пустое количество считает за 1.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID, ok := venueIDFromQuery(w, r)
	if !ok {
		return
	}

	var req domain.AddToCartRequest
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}

	if !checkUUID(w, r, "menu_item_id", req.MenuItemID) {
		return
	}

	if req.Quantity == 0 {
		req.Quantity = 1
	}

	cart, err := h.cartService.AddItem(r.Context(), userID, venueID, req)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

// UpdateItemQuantity - меняет количество у позиции корзины.
func (h *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	itemID, ok := idFromPath(w, r)
	if !ok {
		return
	}

	var req struct {
		Quantity int `json:"quantity"` // новое количество
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}

	if req.Quantity <= 0 {
		writeError(w, r, http.StatusBadRequest, "quantity must be > 0")
		return
	}

	if err := h.cartService.UpdateItemQuantity(r.Context(), userID, itemID, req.Quantity); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "item updated"})
}

// RemoveItem - убирает позицию из корзины.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	itemID, ok := idFromPath(w, r)
	if !ok {
		return
	}

	if err := h.cartService.RemoveItem(r.Context(), userID, itemID); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "item removed"})
}

// Clear - очищает корзину пользователя в заведении.
func (h *CartHandler) Clear(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID, ok := venueIDFromQuery(w, r)
	if !ok {
		return
	}

	if err := h.cartService.Clear(r.Context(), userID, venueID); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "cart cleared"})
}
