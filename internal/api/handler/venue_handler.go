package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/domain"
)

// VenueHandler - HTTP для заведений, публичная часть.
type VenueHandler struct {
	venueService domain.VenueService // логика заведений
}

// NewVenueHandler - создаёт хендлер заведений.
func NewVenueHandler(venueService domain.VenueService) *VenueHandler {
	return &VenueHandler{venueService: venueService}
}

// List - отдаёт список заведений с поиском и пагинацией.
func (h *VenueHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, perPage := paginationFromQuery(r)

	filters := domain.VenueFilters{
		Search:  search,
		Page:    page,
		PerPage: perPage,
	}

	venues, total, err := h.venueService.List(r.Context(), filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, venues, page, perPage, total)
}

// GetByID - отдаёт одно заведение по id из пути.
func (h *VenueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	venue, err := h.venueService.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": venue})
}
