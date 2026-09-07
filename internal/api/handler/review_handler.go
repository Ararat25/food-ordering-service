package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/domain"
)

// ReviewHandler - HTTP для отзывов.
type ReviewHandler struct {
	reviewService domain.ReviewService // логика отзывов
}

// NewReviewHandler - создаёт хендлер отзывов.
func NewReviewHandler(reviewService domain.ReviewService) *ReviewHandler {
	return &ReviewHandler{reviewService: reviewService}
}

// Create - оставляет отзыв о заведении по id из пути.
func (h *ReviewHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	venueID := chi.URLParam(r, "id")

	var req struct {
		OrderID string `json:"order_id"` // по какому заказу отзыв
		Rating  int    `json:"rating"`   // оценка от 1 до 5
		Comment string `json:"comment"`  // текст отзыва
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	review, err := h.reviewService.Create(r.Context(), userID, venueID, req.OrderID, domain.CreateReviewRequest{
		Rating:  req.Rating,
		Comment: req.Comment,
	})
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": review})
}

// ListByVenue - отдаёт отзывы заведения постранично.
func (h *ReviewHandler) ListByVenue(w http.ResponseWriter, r *http.Request) {
	venueID := chi.URLParam(r, "id")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	page, perPage = domain.NormalizePagination(page, perPage)

	reviews, total, err := h.reviewService.ListByVenue(r.Context(), venueID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, reviews, page, perPage, total)
}
