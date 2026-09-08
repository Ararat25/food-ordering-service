package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"avito-kitchen/internal/config"
	"avito-kitchen/internal/domain"
)

// OrderStatusUpdate - смена статуса заказа для вебхука.
type OrderStatusUpdate struct {
	OrderID string `json:"order_id"` // какой заказ
	Status  string `json:"status"`   // новый статус
}

// apiClient - клиент основного сервиса.
type apiClient struct {
	baseURL string       // адрес основного сервиса
	apiKey  string       // ключ заведения
	client  *http.Client // HTTP-клиент с таймаутом
}

// newAPIClient - создаёт клиента к основному сервису.
func newAPIClient(baseURL, apiKey string, timeout time.Duration) *apiClient {
	return &apiClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: timeout},
	}
}

// do - делает запрос с API-ключом в заголовке.
func (c *apiClient) do(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	return c.client.Do(req)
}

// venueOrder - заказ заведения.
type venueOrder struct {
	ID     string `json:"id"`     // id заказа
	Status string `json:"status"` // текущий статус
}

// ordersResponse - ответ списка заказов.
type ordersResponse struct {
	Data []venueOrder `json:"data"` // сами заказы
}

// listOrders - забирает заказы заведения из основного сервиса.
func (c *apiClient) listOrders(ctx context.Context) ([]venueOrder, error) {
	resp, err := c.do(ctx, http.MethodGet, "/api/v1/venue/orders", nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list orders: unexpected status %s", resp.Status)
	}

	var out ordersResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode orders: %w", err)
	}
	return out.Data, nil
}

// acceptOrder - принимает заказ в основном сервисе.
func (c *apiClient) acceptOrder(ctx context.Context, orderID string) error {
	resp, err := c.do(ctx, http.MethodPatch, "/api/v1/venue/orders/"+orderID+"/accept", bytes.NewReader([]byte("{}")))
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4<<10))

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("accept order %s: unexpected status %s", orderID, resp.Status)
	}
	return nil
}

// orderSyncer - умеет забирать заказы и принимать их.
type orderSyncer interface {
	listOrders(ctx context.Context) ([]venueOrder, error)
	acceptOrder(ctx context.Context, orderID string) error
}

// syncOnce - забирает заказы и принимает первый ожидающий.
func syncOnce(ctx context.Context, logger *slog.Logger, client orderSyncer) {
	orders, err := client.listOrders(ctx)
	if err != nil {
		logger.Warn("venue sync: failed to fetch orders", "error", err)
		return
	}

	pending := 0
	for _, o := range orders {
		if o.Status == string(domain.OrderStatusPending) {
			pending++
		}
	}
	logger.Info("venue sync: orders fetched", "total", len(orders), "pending", pending)

	for _, o := range orders {
		if o.Status != string(domain.OrderStatusPending) {
			continue
		}
		if err := client.acceptOrder(ctx, o.ID); err != nil {
			logger.Warn("venue sync: failed to accept order", "order_id", o.ID, "error", err)
			return
		}
		logger.Info("venue sync: order accepted", "order_id", o.ID)
		return
	}
}

// pollLoop - крутит синхронизацию по таймеру, пока не отменят контекст.
func pollLoop(ctx context.Context, logger *slog.Logger, client orderSyncer, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	syncOnce(ctx, logger, client)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			syncOnce(ctx, logger, client)
		}
	}
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.LoadVenue()
	if err != nil {
		logger.Error("venue service failed to load config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if cfg.MainAPIURL == "" || cfg.APIKey == "" {
		logger.Warn("venue sync disabled: MAIN_API_URL or VENUE_API_KEY is not set")
	} else {
		go pollLoop(ctx, logger, newAPIClient(cfg.MainAPIURL, cfg.APIKey, cfg.RequestTimeout), cfg.PollInterval)
	}

	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"venue"}`))
	})

	mux.HandleFunc("GET /api/v1/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[],"message":"venue orders endpoint"}`))
	})

	mux.HandleFunc("POST /webhook/order-status", func(w http.ResponseWriter, r *http.Request) {
		var update OrderStatusUpdate
		if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
			http.Error(w, `{"error":"invalid request"}`, http.StatusBadRequest)
			return
		}

		logger.Info("order status update",
			"order_id", update.OrderID,
			"status", update.Status,
		)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"message":"status received"}`))
	})

	srv := &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: mux,
	}

	go func() {
		logger.Info("venue service starting", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("venue service failed", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	logger.Info("shutting down venue service...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("venue service forced to shutdown", "error", err)
	}

	logger.Info("venue service exited")
}
