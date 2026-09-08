package handler

import (
	"log/slog"
	"net/http"

	"avito-kitchen/internal/domain"
)

// VenueHandler - HTTP для заведений, публичная часть.
type VenueHandler struct {
	venueService domain.VenueService // логика заведений
	logger       *slog.Logger        // куда писать внутренние ошибки
}

// NewVenueHandler - создаёт хендлер заведений.
func NewVenueHandler(venueService domain.VenueService, logger *slog.Logger) *VenueHandler {
	return &VenueHandler{venueService: venueService, logger: logger}
}

// List - отдаёт список заведений с поиском и пагинацией.
func (h *VenueHandler) List(w http.ResponseWriter, r *http.Request) {
	search := r.URL.Query().Get("search")
	page, perPage, err := paginationFromQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	filters := domain.VenueFilters{
		Search:  search,
		Page:    page,
		PerPage: perPage,
	}

	venues, total, err := h.venueService.List(r.Context(), filters)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, venues, page, perPage, total)
}

// GetByID - отдаёт одно заведение по id из пути.
func (h *VenueHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := idFromPath(w, r)
	if !ok {
		return
	}

	venue, err := h.venueService.GetByID(r.Context(), id)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": venue})
}
