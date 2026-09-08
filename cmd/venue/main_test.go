package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

// stubSyncer - заглушка orderSyncer: отдаёт заранее заданные заказы и пишет принятые.
type stubSyncer struct {
	orders   []venueOrder // что вернёт listOrders
	listErr  error        // ошибка listOrders, если нужна
	accepted []string     // какие заказы приняли
}

func (s *stubSyncer) listOrders(_ context.Context) ([]venueOrder, error) {
	return s.orders, s.listErr
}

func (s *stubSyncer) acceptOrder(_ context.Context, orderID string) error {
	s.accepted = append(s.accepted, orderID)
	return nil
}

// quietLogger - логгер, молча глотающий вывод тестов.
func quietLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// TestSyncOnceAcceptsPending - первый ожидающий заказ принимают.
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

// TestSyncOnceWithoutPending - без ожидающих никого не трогают.
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

// TestSyncOnceFetchError - ошибка получения заказов не роняет синхронизацию.
func TestSyncOnceFetchError(t *testing.T) {
	stub := &stubSyncer{listErr: errors.New("connection refused")}

	syncOnce(context.Background(), quietLogger(), stub)

	if len(stub.accepted) != 0 {
		t.Fatalf("want no accepted orders, got %v", stub.accepted)
	}
}
