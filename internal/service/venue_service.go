package service

import (
	"context"

	"avito-kitchen/internal/domain"
)

// VenueService - логика заведений.
type VenueService struct {
	venueRepo domain.VenueRepository // заведения в базе
}

// NewVenueService - создаёт сервис заведений.
func NewVenueService(venueRepo domain.VenueRepository) *VenueService {
	return &VenueService{venueRepo: venueRepo}
}

// GetByID - возвращает заведение по id.
func (s *VenueService) GetByID(ctx context.Context, id string) (*domain.Venue, error) {
	return s.venueRepo.GetByID(ctx, id)
}

// GetByAPIKey - возвращает заведение по API-ключу, нужно для auth заведений.
func (s *VenueService) GetByAPIKey(ctx context.Context, apiKey string) (*domain.Venue, error) {
	return s.venueRepo.GetByAPIKey(ctx, apiKey)
}

// List - возвращает заведения с фильтрами и пагинацией.
func (s *VenueService) List(ctx context.Context, filters domain.VenueFilters) ([]domain.Venue, int, error) {
	filters.Page, filters.PerPage = domain.NormalizePagination(filters.Page, filters.PerPage)
	return s.venueRepo.List(ctx, filters)
}

// Create - создаёт заведение.
func (s *VenueService) Create(ctx context.Context, venue *domain.Venue) error {
	return s.venueRepo.Create(ctx, venue)
}
