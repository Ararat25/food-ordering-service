package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// MenuHandler - HTTP для меню, публичная часть и кабинет заведения.
type MenuHandler struct {
	menuService domain.MenuService // логика меню
}

// NewMenuHandler - создаёт хендлер меню.
func NewMenuHandler(menuService domain.MenuService) *MenuHandler {
	return &MenuHandler{menuService: menuService}
}

// ListItems - отдаёт блюда заведения из контекста постранично, для кабинета.
func (h *MenuHandler) ListItems(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())
	page, perPage := paginationFromQuery(r)

	items, total, err := h.menuService.ListItems(r.Context(), venueID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, items, page, perPage, total)
}

// ListByVenuePublic - отдаёт блюда заведения по id из пути постранично, для всех.
func (h *MenuHandler) ListByVenuePublic(w http.ResponseWriter, r *http.Request) {
	venueID := chi.URLParam(r, "id")
	page, perPage := paginationFromQuery(r)

	items, total, err := h.menuService.ListItems(r.Context(), venueID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, items, page, perPage, total)
}

// CreateItem - создаёт блюдо в своём заведении.
func (h *MenuHandler) CreateItem(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())

	var item domain.MenuItem
	if err := decodeJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item.VenueID = venueID

	if err := h.menuService.CreateItem(r.Context(), &item); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": item})
}

// UpdateItem - обновляет блюдо своего заведения.
func (h *MenuHandler) UpdateItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	var item domain.MenuItem
	if err := decodeJSON(r, &item); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	item.ID = itemID
	item.VenueID = venueID

	if err := h.menuService.UpdateItem(r.Context(), &item); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"data": item})
}

// DeleteItem - удаляет блюдо своего заведения.
func (h *MenuHandler) DeleteItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "id")
	venueID, _ := middleware.GetVenueID(r.Context())

	if err := h.menuService.DeleteItem(r.Context(), venueID, itemID); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
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
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.menuService.UpdateAvailability(r.Context(), venueID, itemID, req.Available); err != nil {
		writeError(w, domainErrorToHTTP(err), err.Error())
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
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, categories, page, perPage, total)
}

// CreateCategory - создаёт категорию в своём заведении.
func (h *MenuHandler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	venueID, _ := middleware.GetVenueID(r.Context())

	var cat domain.MenuCategory
	if err := decodeJSON(r, &cat); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	cat.VenueID = venueID

	if err := h.menuService.CreateCategory(r.Context(), &cat); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{"data": cat})
}
