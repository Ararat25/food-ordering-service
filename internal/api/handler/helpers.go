package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// paginationFromQuery - достаёт page/per_page из query, кривые значения чинит нормализация.
func paginationFromQuery(r *http.Request) (int, int) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	return domain.NormalizePagination(page, perPage)
}

// writeJSON - отдаёт ответ в JSON с нужным статусом.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError - отдаёт ошибку в формате {"error","code"} плюс request_id для корреляции с логами.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	body := map[string]any{
		"error": message,
		"code":  status,
	}
	if requestID, ok := r.Context().Value(middleware.RequestIDKey).(string); ok {
		body["request_id"] = requestID
	}
	writeJSON(w, status, body)
}

// writeServiceError - отдаёт ошибку сервисного слоя: клиентские как есть,
// внутренние прячет за общим текстом, а детали пишет в лог.
func writeServiceError(w http.ResponseWriter, r *http.Request, logger *slog.Logger, err error) {
	status := domainErrorToHTTP(err)
	if status >= http.StatusInternalServerError {
		logger.Error("internal error",
			"error", err,
			"method", r.Method,
			"path", r.URL.Path,
		)
		writeError(w, r, status, "internal error")
		return
	}
	writeError(w, r, status, err.Error())
}

// writePaginated - отдаёт список с блоком пагинации.
func writePaginated(w http.ResponseWriter, data any, page, perPage, total int) {
	writeJSON(w, http.StatusOK, map[string]any{
		"data": data,
		"pagination": map[string]any{
			"page":        page,
			"per_page":    perPage,
			"total":       total,
			"total_pages": domain.TotalPages(total, perPage),
		},
	})
}

// maxBodyBytes - потолок тела запроса, защита от гигантских JSON.
const maxBodyBytes = 1 << 20

// decodeJSON - читает JSON из тела запроса, лишние поля и переполнение отбивает.
func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}

// domainErrorToHTTP - переводит ошибку домена в HTTP-код.
func domainErrorToHTTP(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, domain.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(err, domain.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(err, domain.ErrValidation):
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
