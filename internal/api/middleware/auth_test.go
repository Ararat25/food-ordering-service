package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

type testUserRepo struct {
	exists bool
	err    error
	calls  int
}

func (r *testUserRepo) Exists(context.Context, string) (bool, error) {
	r.calls++
	return r.exists, r.err
}

type testVenueService struct{ err error }

func (s testVenueService) List(context.Context, domain.VenueFilters) ([]domain.Venue, int, error) {
	return nil, 0, s.err
}
func (s testVenueService) GetByID(context.Context, string) (*domain.Venue, error) { return nil, s.err }
func (s testVenueService) GetByAPIKey(context.Context, string) (*domain.Venue, error) {
	return nil, s.err
}
func (s testVenueService) Create(context.Context, *domain.Venue) error { return s.err }

func TestUserAuthChecksUUIDAndExistence(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		exists     bool
		repoErr    error
		wantStatus int
		wantCalls  int
	}{
		{name: "missing", wantStatus: http.StatusUnauthorized},
		{name: "malformed", userID: "not-a-uuid", wantStatus: http.StatusBadRequest},
		{name: "unknown", userID: "550e8400-e29b-41d4-a716-446655440099", wantStatus: http.StatusUnauthorized, wantCalls: 1},
		{name: "database failure", userID: "550e8400-e29b-41d4-a716-446655440001", repoErr: errors.New("db offline"), wantStatus: http.StatusInternalServerError, wantCalls: 1},
		{name: "known", userID: "550e8400-e29b-41d4-a716-446655440001", exists: true, wantStatus: http.StatusNoContent, wantCalls: 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &testUserRepo{exists: tt.exists, err: tt.repoErr}
			h := RequestID(UserAuth(repo)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
			r := httptest.NewRequest(http.MethodGet, "/private", nil)
			if tt.userID != "" {
				r.Header.Set("X-User-ID", tt.userID)
			}
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, tt.wantStatus, w.Code)
			require.Equal(t, tt.wantCalls, repo.calls)
			if tt.wantStatus != http.StatusNoContent {
				require.Equal(t, "application/json", w.Header().Get("Content-Type"))
				require.Contains(t, w.Body.String(), "request_id")
			}
		})
	}
}

func TestVenueAuthDistinguishesInvalidKeyFromRepositoryFailure(t *testing.T) {
	for _, tt := range []struct {
		name       string
		err        error
		wantStatus int
	}{
		{name: "invalid key", err: domain.ErrNotFound, wantStatus: http.StatusUnauthorized},
		{name: "database failure", err: errors.New("db offline"), wantStatus: http.StatusInternalServerError},
	} {
		t.Run(tt.name, func(t *testing.T) {
			h := RequestID(VenueAuth(testVenueService{err: tt.err})(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })))
			r := httptest.NewRequest(http.MethodGet, "/private", nil)
			r.Header.Set("Authorization", "Bearer key")
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			require.Equal(t, tt.wantStatus, w.Code)
			require.Equal(t, "application/json", w.Header().Get("Content-Type"))
			require.True(t, strings.Contains(w.Body.String(), "request_id"))
		})
	}
}
