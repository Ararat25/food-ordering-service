package handler

import (
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// MenuHandler - HTTP для меню, публичная часть и кабинет заведения.
type MenuHandler struct {
	menuService domain.MenuService // логика меню
	logger      *slog.Logger       // куда писать внутренние ошибки
}

// NewMenuHandler - создаёт хендлер меню.
func NewMenuHandler(menuService domain.MenuService, logger *slog.Logger) *MenuHandler {
	return &MenuHandler{menuService: menuService, logger: logger}
}

// menuFiltersFromQuery - собирает фильтры меню из query: поиск, наличие, пагинация.
func menuFiltersFromQuery(r *http.Request, venueID string) domain.MenuItemFilters {
	page, perPage := paginationFromQuery(r)

	var available *bool
	if raw := r.URL.Query().Get("available"); raw != "" {
		if parsed, err := strconv.ParseBool(raw); err == nil {
			available = &parsed
		}
	}

	return domain.MenuItemFilters{
		VenueID:   venueID,
		Search:    r.URL.Query().Get("search"),
		Available: available,
		Page:      page,
		PerPage:   perPage,
	}
}

// ListItems - отдаёт блюда заведения из контекста с фильтрами, для кабинета.
func (h *MenuHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())
	filters := menuFiltersFromQuery(r, venueID)

	items, total, err := h.menuService.ListItems(r.Context(), filters)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, items, filters.Page, filters.PerPage, total)
}

// ListByVenuePublic - отдаёт блюда заведения по id из пути с фильтрами, для всех.
func (h *MenuHandler) ListByVenuePublic(w http.ResponseWriter, r *http.Request) {
	venueID := chi.URLParam(r, "id")
	filters := menuFiltersFromQuery(r, venueID)

	items, total, err := h.menuService.ListItems(r.Context(), filters)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, items, filters.Page, filters.PerPage, total)
}

// CreateItem - создаёт блюдо в своём заведении.
func (h *MenuHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())

	var item domain.MenuItem
	if err := decodeJSON(w, r, &item); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	item.VenueID = venueID

	if err := h.menuService.CreateItem(r.Context(), &item); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

// UpdateItem - обновляет блюдо своего заведения.
func (h *MenuHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	var item domain.MenuItem
	if err := decodeJSON(w, r, &item); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	item.ID = itemID
	item.VenueID = venueID

	if err := h.menuService.UpdateItem(r.Context(), &item); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// DeleteItem - удаляет блюдо своего заведения.
func (h *MenuHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.menuService.DeleteItem(r.Context(), venueID, itemID); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "item deleted"})
}

// UpdateAvailability - меняет наличие блюда своего заведения.
func (h *MenuHandler) UpdateAvailability(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	var req struct {
		Available bool `json:"available"` // новое наличие
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.menuService.UpdateAvailability(r.Context(), venueID, itemID, req.Available); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "availability updated"})
}

// ListCategories - отдаёт категории своего заведения постранично.
func (h *MenuHandler) ListCategories(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())
	page, perPage := paginationFromQuery(r)

	categories, total, err := h.menuService.ListCategories(r.Context(), venueID, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, categories, page, perPage, total)
}

// CreateCategory - создаёт категорию в своём заведении.
func (h *MenuHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())

	var cat domain.MenuCategory
	if err := decodeJSON(w, r, &cat); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	cat.VenueID = venueID

	if err := h.menuService.CreateCategory(r.Context(), &cat); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": cat})
}
