package handler

import (
	"log/slog"
	"net/http"
	"unicode/utf8"

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
		writeDecodeError(w, r, err)
		return
	}

	if req.Token == "" || utf8.RuneCountInString(req.Token) > 500 {
		writeError(w, r, http.StatusBadRequest, "token must contain 1 to 500 characters")
		return
	}
	switch req.Platform {
	case "ios", "android", "web":
	default:
		writeError(w, r, http.StatusBadRequest, "platform must be ios, android or web")
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

// DeleteToken - отзывает зарегистрированный push token пользователя.
func (h *NotificationHandler) DeleteToken(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}
	var req struct {
		Token string `json:"token"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, r, err)
		return
	}
	if req.Token == "" || utf8.RuneCountInString(req.Token) > 500 {
		writeError(w, r, http.StatusBadRequest, "token must contain 1 to 500 characters")
		return
	}
	if err := h.notifService.UnregisterPushToken(r.Context(), userID, req.Token); err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"message": "token unregistered"})
}

// List - отдаёт уведомления пользователя постранично.
func (h *NotificationHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("X-User-ID")
	if userID == "" {
		writeError(w, r, http.StatusUnauthorized, "X-User-ID header required")
		return
	}

	page, perPage, err := paginationFromQuery(r)
	if err != nil {
		writeError(w, r, http.StatusBadRequest, err.Error())
		return
	}

	notifications, total, err := h.notifService.ListByUser(r.Context(), userID, page, perPage)
	if err != nil {
		writeServiceError(w, r, h.logger, err)
		return
	}

	writePaginated(w, notifications, page, perPage, total)
}
