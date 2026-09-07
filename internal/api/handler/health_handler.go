package handler

import (
	"database/sql"
	"net/http"
)

// HealthHandler - проверки здоровья сервиса.
type HealthHandler struct {
	db *sql.DB // соединение с базой
}

// NewHealthHandler - создаёт хендлер проверок.
func NewHealthHandler(db *sql.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// Health - отвечает ok, если процесс жив.
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

// Ready - отвечает ok, если жив и база отвечает, иначе not ready.
func (h *HealthHandler) Ready(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{
			"status": "not ready",
			"reason": "database connection failed",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}
