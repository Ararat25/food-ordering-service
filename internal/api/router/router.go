package router

import (
	"database/sql"
	"log/slog"

	"github.com/go-chi/chi/v5"

	"avito-kitchen/internal/api/handler"
	"avito-kitchen/internal/api/middleware"
	"avito-kitchen/internal/repository"
	"avito-kitchen/internal/service"
)

// New - собирает все зависимости и возвращает роутер со всеми маршрутами.
func New(db *sql.DB, logger *slog.Logger) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logging(logger))
	r.Use(middleware.CORS)

	venueRepo := repository.NewVenueRepo(db)
	menuCategoryRepo := repository.NewMenuCategoryRepo(db)
	menuItemRepo := repository.NewMenuItemRepo(db)
	cartRepo := repository.NewCartRepo(db)
	orderRepo := repository.NewOrderRepo(db)
	reviewRepo := repository.NewReviewRepo(db)
	notifRepo := repository.NewNotificationRepo(db)
	pushTokenRepo := repository.NewPushTokenRepo(db)

	venueService := service.NewVenueService(venueRepo)
	menuService := service.NewMenuService(menuCategoryRepo, menuItemRepo)
	cartService := service.NewCartService(cartRepo, menuItemRepo)
	orderService := service.NewOrderService(orderRepo, cartRepo, menuItemRepo)
	reviewService := service.NewReviewService(reviewRepo, orderRepo)
	notifService := service.NewNotificationService(notifRepo, pushTokenRepo)

	healthHandler := handler.NewHealthHandler(db)
	venueHandler := handler.NewVenueHandler(venueService)
	cartHandler := handler.NewCartHandler(cartService)
	orderHandler := handler.NewOrderHandler(orderService)
	menuHandler := handler.NewMenuHandler(menuService)
	reviewHandler := handler.NewReviewHandler(reviewService)
	notifHandler := handler.NewNotificationHandler(notifService)

	r.Get("/health", healthHandler.Health)
	r.Get("/health/ready", healthHandler.Ready)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/venues", venueHandler.List)
		r.Get("/venues/{id}", venueHandler.GetByID)
		r.Get("/venues/{id}/menu", menuHandler.ListByVenuePublic)
		r.Get("/venues/{id}/reviews", reviewHandler.ListByVenue)
		r.Post("/venues/{id}/reviews", reviewHandler.Create)

		r.Route("/cart", func(r chi.Router) {
			r.Use(middleware.UserAuth)
			r.Get("/", cartHandler.Get)
			r.Post("/items", cartHandler.AddItem)
			r.Put("/items/{id}", cartHandler.UpdateItemQuantity)
			r.Delete("/items/{id}", cartHandler.RemoveItem)
			r.Delete("/", cartHandler.Clear)
		})

		r.Route("/orders", func(r chi.Router) {
			r.Use(middleware.UserAuth)
			r.Post("/", orderHandler.Create)
			r.Get("/", orderHandler.List)
			r.Get("/{id}", orderHandler.GetByID)
			r.Get("/{id}/track", orderHandler.Track)
			r.Post("/{id}/repeat", orderHandler.Repeat)
		})

		r.Route("/notifications", func(r chi.Router) {
			r.Use(middleware.UserAuth)
			r.Post("/register", notifHandler.RegisterToken)
			r.Get("/", notifHandler.List)
		})
	})

	r.Route("/api/v1/venue", func(r chi.Router) {
		r.Use(middleware.VenueAuth(venueService))

		r.Route("/menu", func(r chi.Router) {
			r.Get("/", menuHandler.ListItems)
			r.Post("/", menuHandler.CreateItem)
			r.Put("/{id}", menuHandler.UpdateItem)
			r.Delete("/{id}", menuHandler.DeleteItem)
			r.Patch("/{id}/availability", menuHandler.UpdateAvailability)
		})

		r.Route("/orders", func(r chi.Router) {
			r.Get("/", orderHandler.VenueList)
			r.Get("/{id}", orderHandler.GetByID)
			r.Patch("/{id}/accept", orderHandler.VenueAccept)
			r.Patch("/{id}/reject", orderHandler.VenueReject)
			r.Patch("/{id}/status", orderHandler.VenueUpdateStatus)
		})
	})

	return r
}
