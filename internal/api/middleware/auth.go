package middleware

import (
	"context"
	"net/http"
	"strings"

	"avito-kitchen/internal/domain"
)

// VenueAuth - проверяет API-ключ заведения и кладёт его id в контекст.
func VenueAuth(venueService domain.VenueService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Authorization header required","code":401}`, http.StatusUnauthorized)
				return
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error":"Invalid authorization format","code":401}`, http.StatusUnauthorized)
				return
			}

			apiKey := parts[1]
			venue, err := venueService.GetByAPIKey(r.Context(), apiKey)
			if err != nil {
				http.Error(w, `{"error":"Invalid API key","code":401}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKey("venue_id"), venue.ID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserAuth - проверяет заголовок пользователя и кладёт его id в контекст.
func UserAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			http.Error(w, `{"error":"X-User-ID header required","code":401}`, http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), contextKey("user_id"), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
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
