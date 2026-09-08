package handler

import (
	"log/slog"
	"net/http"

	"avito-kitchen/internal/domain"
)

// NotificationHandler - обработчик для уведомлений.
type NotificationHandler struct {
	notifService domain.NotificationService // логика уведомлений
	logger       *slog.Logger               // куда писать внутренние ошибки
}

// NewNotificationHandler - создаёт хендлер уведомлений.
func NewNotificationHandler(notifService domain.NotificationService, logger *slog.Logger) *NotificationHandler {
	return &NotificationHandler{notifService: notifService, logger: logger}
}

// RegisterToken - сохраняет пуш-токен устройства пользователя.
func (h *NotificationHandler) RegisterToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	var req struct {
		Token    string `json:"token"`    // сам токен
		Platform string `json:"platform"` // платформа
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, r, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Token == "" || req.Platform == "" {
		writeError(w, r, http.StatusBadRequest, "token and platform are required")
		return
	}

	if err := h.notifService.RegisterPushToken(r.Context(), &domain.PushToken{
		UserID:   userID,
		Token:    req.Token,
		Platform: req.Platform,
	}); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"message": "token registered"})
}

// List - отдаёт уведомления пользователя постранично.
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	page, perPage := paginationFromQuery(r)

	notifications, total, err := h.notifService.ListByUser(r.Context(), userID, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, notifications, page, perPage, total)
}
