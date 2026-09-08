package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/domain"
)

// paginationFromQuery - defaults omitted values and rejects malformed/out-of-range values.
func paginationFromQuery(r *http.Request) (int, int, error) {
	page, perPage := domain.DefaultPage, domain.DefaultPerPage
	if raw := r.URL.Query().Get("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 {
			return 0, 0, fmt.Errorf("%w: page must be a positive integer", domain.ErrValidation)
		}
		page = parsed
	}
	if raw := r.URL.Query().Get("per_page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed <= 0 || parsed > domain.MaxPerPage {
			return 0, 0, fmt.Errorf("%w: per_page must be between 1 and %d", domain.ErrValidation, domain.MaxPerPage)
		}
		perPage = parsed
	}
	maxInt := int(^uint(0) >> 1)
	if page-1 > maxInt/perPage {
		return 0, 0, fmt.Errorf("%w: page is too large", domain.ErrValidation)
	}
	return page, perPage, nil
}

// writeJSON - отдаёт ответ в JSON с нужным статусом.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError - отдаёт ошибку в формате {"error","code"} плюс request_id для корреляции с логами.
func writeError(w http.ResponseWriter, r *http.Request, status int, message string) {
	middleware.WriteError(w, r, status, message)
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
			"request_id", r.Context().Value(middleware.RequestIDKey),
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
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("request body must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func writeDecodeError(w http.ResponseWriter, r *http.Request, err error) {
	var tooLarge *http.MaxBytesError
	if errors.As(err, &tooLarge) {
		writeError(w, r, http.StatusRequestEntityTooLarge, "request body too large")
		return
	}
	writeError(w, r, http.StatusBadRequest, "invalid request body")
}

// domainErrorToHTTP - переводит ошибку домена в HTTP-код.
func domainErrorToHTTP(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, domain.ErrConflict):
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
