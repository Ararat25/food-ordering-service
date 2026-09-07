package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// ReviewService - логика отзывов.
type ReviewService struct {
	reviewRepo domain.ReviewRepository // отзывы в базе
	orderRepo  domain.OrderRepository  // заказы в базе
}

// NewReviewService - создаёт сервис отзывов.
func NewReviewService(reviewRepo domain.ReviewRepository, orderRepo domain.OrderRepository) *ReviewService {
	return &ReviewService{reviewRepo: reviewRepo, orderRepo: orderRepo}
}

// Create - оставляет отзыв, можно только по своему доставленному заказу и только один раз.
func (s *ReviewService) Create(ctx context.Context, userID, venueID, orderID string, req domain.CreateReviewRequest) (*domain.Review, error) {
	if req.Rating < 1 || req.Rating > 5 {
		return nil, fmt.Errorf("%w: rating must be between 1 and 5", domain.ErrValidation)
	}

	order, err := s.orderRepo.GetByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("order not found: %w", err)
	}

	if order.UserID != userID {
		return nil, domain.ErrForbidden
	}

	if order.VenueID != venueID {
		return nil, fmt.Errorf("%w: order does not belong to this venue", domain.ErrValidation)
	}

	if order.Status != domain.OrderStatusDelivered {
		return nil, fmt.Errorf("%w: can only review delivered orders", domain.ErrValidation)
	}

	existing, err := s.reviewRepo.GetByUserAndOrder(ctx, userID, orderID)
	if err != nil && !errors.Is(err, domain.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return nil, domain.ErrAlreadyExists
	}

	review := &domain.Review{
		ID:      uuid.New().String(),
		UserID:  userID,
		VenueID: venueID,
		OrderID: orderID,
		Rating:  req.Rating,
		Comment: req.Comment,
	}

	if err := s.reviewRepo.Create(ctx, review); err != nil {
		return nil, err
	}

	return review, nil
}

// ListByVenue - возвращает отзывы заведения постранично.
func (s *ReviewService) ListByVenue(ctx context.Context, venueID string, page, perPage int) ([]domain.Review, int, error) {
	page, perPage = domain.NormalizePagination(page, perPage)
	return s.reviewRepo.ListByVenue(ctx, venueID, page, perPage)
}
