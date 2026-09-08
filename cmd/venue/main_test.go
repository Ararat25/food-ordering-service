package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"avito-kitchen/internal/api/middleware"
)

type stubSyncer struct {
	orders   []venueOrder
	listErr  error
	accepted []string
}

func (s *stubSyncer) listOrders(_ context.Context) ([]venueOrder, error) {
	return s.orders, s.listErr
}

func (s *stubSyncer) acceptOrder(_ context.Context, orderID string) error {
	s.accepted = append(s.accepted, orderID)
	return nil
}

func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestSyncOnceAcceptsPending(t *testing.T) {
	stub := &stubSyncer{orders: []venueOrder{
		{ID: "order-1", Status: "accepted"},
		{ID: "order-2", Status: "pending"},
		{ID: "order-3", Status: "pending"},
	}}

	syncOnce(context.Background(), quietLogger(), stub)

	if len(stub.accepted) != 1 || stub.accepted[0] != "order-2" {
		t.Fatalf("want exactly [order-2] accepted, got %v", stub.accepted)
	}
}

func TestSyncOnceWithoutPending(t *testing.T) {
	stub := &stubSyncer{orders: []venueOrder{
		{ID: "order-1", Status: "accepted"},
		{ID: "order-2", Status: "delivered"},
	}}

	syncOnce(context.Background(), quietLogger(), stub)

	if len(stub.accepted) != 0 {
		t.Fatalf("want no accepted orders, got %v", stub.accepted)
	}
}

func TestSyncOnceFetchError(t *testing.T) {
	stub := &stubSyncer{listErr: errors.New("connection refused")}

	syncOnce(context.Background(), quietLogger(), stub)

	if len(stub.accepted) != 0 {
		t.Fatalf("want no accepted orders, got %v", stub.accepted)
	}
}

func TestOrderStatusWebhookUsesJSONErrorsAndLimitsBody(t *testing.T) {
	h := middleware.RequestID(orderStatusWebhookHandler(quietLogger()))
	for _, tt := range []struct {
		name       string
		body       string
		wantStatus int
	}{
		{name: "valid", body: `{"order_id":"order-1","status":"ready"}`, wantStatus: http.StatusOK},
		{name: "trailing value", body: `{"order_id":"order-1","status":"ready"} {}`, wantStatus: http.StatusBadRequest},
		{name: "oversized", body: `{"order_id":"` + strings.Repeat("x", 1<<20) + `","status":"ready"}`, wantStatus: http.StatusRequestEntityTooLarge},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, "/webhook/order-status", strings.NewReader(tt.body))
			w := httptest.NewRecorder()
			h.ServeHTTP(w, r)
			if w.Code != tt.wantStatus {
				t.Fatalf("want status %d, got %d: %s", tt.wantStatus, w.Code, w.Body.String())
			}
			if tt.wantStatus != http.StatusOK {
				if w.Header().Get("Content-Type") != "application/json" || !strings.Contains(w.Body.String(), "request_id") {
					t.Fatalf("expected shared JSON error envelope, got headers=%v body=%s", w.Header(), w.Body.String())
				}
			}
		})
	}
}
