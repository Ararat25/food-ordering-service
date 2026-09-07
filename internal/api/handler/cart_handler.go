package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/domain"
)

// CartHandler - HTTP для корзины.
type CartHandler struct {
	cartService domain.CartService // логика корзины
}

// NewCartHandler - создаёт хендлер корзины.
func NewCartHandler(cartService domain.CartService) *CartHandler {
	return &CartHandler{cartService: cartService}
}

// Get - отдаёт корзину пользователя в заведении из query.
func (h *CartHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID := r.URL.Query().Get("venue_id")
	if venueID == "" {
		writeError(w, http.StatusBadRequest, "venue_id query parameter required")
		return
	}

	cart, err := h.cartService.GetCart(r.Context(), userID, venueID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

// AddItem - кладёт блюдо в корзину, пустое количество считает за 1.
func (h *CartHandler) AddItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID := r.URL.Query().Get("venue_id")
	if venueID == "" {
		writeError(w, http.StatusBadRequest, "venue_id query parameter required")
		return
	}

	var req domain.AddToCartRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Quantity <= 0 {
		req.Quantity = 1
	}

	cart, err := h.cartService.AddItem(r.Context(), userID, venueID, req)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": cart})
}

// UpdateItemQuantity - меняет количество у позиции корзины.
func (h *CartHandler) UpdateItemQuantity(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	itemID := chi.URLParam(r, "id")

	var req struct {
		Quantity int `json:"quantity"` // новое количество
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Quantity <= 0 {
		writeError(w, http.StatusBadRequest, "quantity must be > 0")
		return
	}

	if err := h.cartService.UpdateItemQuantity(r.Context(), userID, itemID, req.Quantity); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "item updated"})
}

// RemoveItem - убирает позицию из корзины.
func (h *CartHandler) RemoveItem(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	itemID := chi.URLParam(r, "id")

	if err := h.cartService.RemoveItem(r.Context(), userID, itemID); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "item removed"})
}

// Clear - очищает корзину пользователя в заведении.
func (h *CartHandler) Clear(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID := r.URL.Query().Get("venue_id")
	if venueID == "" {
		writeError(w, http.StatusBadRequest, "venue_id query parameter required")
		return
	}

	if err := h.cartService.Clear(r.Context(), userID, venueID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "cart cleared"})
}
