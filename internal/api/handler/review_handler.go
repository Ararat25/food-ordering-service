package handler

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/domain"
)

// ReviewHandler - HTTP для отзывов.
type ReviewHandler struct {
	reviewService domain.ReviewService // логика отзывов
	logger        *slog.Logger         // куда писать внутренние ошибки
}

// NewReviewHandler - создаёт хендлер отзывов.
func NewReviewHandler(reviewService domain.ReviewService, logger *slog.Logger) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService, logger: logger}
}

// Create - оставляет отзыв о заведении по id из пути.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID := chi.URLParam(r, "id")

	var req struct {
		OrderID string `json:"order_id"` // по какому заказу отзыв
		Rating  int    `json:"rating"`   // оценка
		Comment string `json:"comment"`  // текст отзыва
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}

	review, err := h.reviewService.Create(r.Context(), userID, venueID, req.OrderID, domain.CreateReviewRequest{
		Rating:  req.Rating,
		Comment: req.Comment,
	})
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": review})
}

// ListByVenue - отдаёт отзывы заведения постранично.
func (h *ReviewHandler) ListByVenue(w http.ResponseWriter, r *http.Request) {
	venueID := chi.URLParam(r, "id")
	page, perPage, err := paginationFromQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	reviews, total, err := h.reviewService.ListByVenue(r.Context(), venueID, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, reviews, page, perPage, total)
}
