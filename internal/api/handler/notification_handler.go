package handler

import (
	"net/http"
	"strconv"

	"avito-kitchen/internal/domain"
)

// NotificationHandler - HTTP для уведомлений.
type NotificationHandler struct {
	notifService domain.NotificationService // логика уведомлений
}

// NewNotificationHandler - создаёт хендлер уведомлений.
func NewNotificationHandler(notifService domain.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifService: notifService}
}

// RegisterToken - сохраняет пуш-токен устройства пользователя.
func (h *NotificationHandler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	var req struct {
		Token    string `json:"token"`    // сам токен
		Platform string `json:"platform"` // платформа, например ios
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Platform == "" {
		writeError(w, http.StatusBadRequest, "token and platform are required")
		return
	}

	if err := h.notifService.RegisterPushToken(r.Context(), &domain.PushToken{
		UserID:   userID,
		Token:    req.Token,
		Platform: req.Platform,
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "token registered"})
}

// List - отдаёт уведомления пользователя постранично.
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	page, perPage = domain.NormalizePagination(page, perPage)

	notifications, total, err := h.notifService.ListByUser(r.Context(), userID, page, perPage)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writePaginated(w, notifications, page, perPage, total)
}
