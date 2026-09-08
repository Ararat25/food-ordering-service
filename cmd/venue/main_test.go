package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
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
