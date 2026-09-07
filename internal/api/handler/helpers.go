package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"avito-kitchen/internal/domain"
)

// writeJSON - отдаёт ответ в JSON с нужным статусом.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError - отдаёт ошибку в формате {"error","code"}.
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{
		"error": message,
		"code":  status,
	})
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

// decodeJSON - читает JSON из тела запроса.
func decodeJSON(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
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
