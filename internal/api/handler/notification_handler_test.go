package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

type stubNotificationService struct {
	unregisterErr error
	registered    int
	unregistered  []string
}

func (s *stubNotificationService) Create(context.Context, *domain.Notification) error { return nil }
func (s *stubNotificationService) ListByUser(context.Context, string, int, int) ([]domain.Notification, int, error) {
	return nil, 0, nil
}
func (s *stubNotificationService) MarkAsRead(context.Context, string) error { return nil }
func (s *stubNotificationService) RegisterPushToken(context.Context, *domain.PushToken) error {
	s.registered++
	return nil
}
func (s *stubNotificationService) UnregisterPushToken(_ context.Context, userID, token string) error {
	s.unregistered = append(s.unregistered, userID+":"+token)
	return s.unregisterErr
}

func TestNotificationDeleteTokenIsUserScoped(t *testing.T) {
	svc := &stubNotificationService{}
	h := NewNotificationHandler(svc, testLogger())
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/register", strings.NewReader(`{"token":"device-token"}`))
	r.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	h.DeleteToken(w, withRequestID(r))

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"user-1:device-token"}, svc.unregistered)

	svc.unregisterErr = domain.ErrNotFound
	r = httptest.NewRequest(http.MethodDelete, "/api/v1/notifications/register", strings.NewReader(`{"token":"device-token"}`))
	r.Header.Set("X-User-ID", "user-2")
	w = httptest.NewRecorder()
	h.DeleteToken(w, withRequestID(r))
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestNotificationRegisterRejectsUnsupportedPlatform(t *testing.T) {
	svc := &stubNotificationService{}
	h := NewNotificationHandler(svc, testLogger())
	r := httptest.NewRequest(http.MethodPost, "/api/v1/notifications/register", strings.NewReader(`{"token":"device-token","platform":"unknown"}`))
	r.Header.Set("X-User-ID", "user-1")
	w := httptest.NewRecorder()
	h.RegisterToken(w, withRequestID(r))

	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Zero(t, svc.registered)
}
