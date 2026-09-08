package middleware

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// VenueAuth - проверяет API-ключ заведения и кладёт его id в контекст.
func VenueAuth(venueService domain.VenueService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				WriteError(w, r, http.StatusUnauthorized, "Authorization header required")
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				WriteError(w, r, http.StatusUnauthorized, "Invalid authorization format")
				return
			}

			apiKey := parts[1]
			venue, err := venueService.GetByAPIKey(r.Context(), apiKey)
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) || errors.Is(err, domain.ErrUnauthorized) {
					WriteError(w, r, http.StatusUnauthorized, "Invalid API key")
					return
				}
				slog.ErrorContext(r.Context(), "venue authentication lookup failed", "error", err, "request_id", r.Context().Value(RequestIDKey))
				WriteError(w, r, http.StatusInternalServerError, "internal error")
				return
			}

			ctx := context.WithValue(r.Context(), contextKey("venue_id"), venue.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserAuth - проверяет заголовок пользователя и кладёт его id в контекст.
func UserAuth(userRepo domain.UserIdentityRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := r.Header.Get("X-User-ID")
			if userID == "" {
				WriteError(w, r, http.StatusUnauthorized, "X-User-ID header required")
				return
			}
			if _, err := uuid.Parse(userID); err != nil {
				WriteError(w, r, http.StatusBadRequest, "X-User-ID must be a UUID")
				return
			}
			exists, err := userRepo.Exists(r.Context(), userID)
			if err != nil {
				slog.ErrorContext(r.Context(), "user identity lookup failed", "error", err, "request_id", r.Context().Value(RequestIDKey))
				WriteError(w, r, http.StatusInternalServerError, "internal error")
				return
			}
			if !exists {
				WriteError(w, r, http.StatusUnauthorized, "unknown X-User-ID")
				return
			}
			ctx := context.WithValue(r.Context(), contextKey("user_id"), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetVenueID - достаёт id заведения из контекста.
func GetVenueID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKey("venue_id")).(string)
	return id, ok
}

// GetUserID - достаёт id пользователя из контекста.
func GetUserID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(contextKey("user_id")).(string)
	return id, ok
}
