package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func reviewTestSetup() (*stubReviewRepo, *stubOrderRepo) {
	reviews := &stubReviewRepo{existErr: domain.ErrNotFound}
	orders := &stubOrderRepo{order: &domain.Order{
		ID: "order-1", UserID: "user-1", VenueID: "venue-1", Status: domain.OrderStatusDelivered,
	}}
	return reviews, orders
}

func TestReviewCreate(t *testing.T) {
	reviews, orders := reviewTestSetup()
	svc := NewReviewService(reviews, orders)

	got, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{
		Rating: 5, Comment: "Вкусно!",
	})
	require.NoError(t, err)
	require.NotEmpty(t, got.ID)
	require.Equal(t, 5, got.Rating)
	require.Equal(t, "user-1", got.UserID)
	require.Len(t, reviews.created, 1)
}

func TestReviewCreateValidation(t *testing.T) {
	for _, rating := range []int{0, -1, 6, 100} {
		reviews, orders := reviewTestSetup()
		svc := NewReviewService(reviews, orders)
		_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{
			Rating: rating,
		})
		require.ErrorIs(t, err, domain.ErrValidation, "rating %d", rating)
		require.Empty(t, reviews.created)
	}
}

func TestReviewCreateRules(t *testing.T) {
	t.Run("заказа нет", func(t *testing.T) {
		reviews, _ := reviewTestSetup()
		orders := &stubOrderRepo{orderErr: domain.ErrNotFound}
		svc := NewReviewService(reviews, orders)
		_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{Rating: 5})
		require.Error(t, err)
		require.Empty(t, reviews.created)
	})

	t.Run("чужой заказ", func(t *testing.T) {
		reviews, orders := reviewTestSetup()
		orders.order.UserID = "user-2"
		svc := NewReviewService(reviews, orders)
		_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{Rating: 5})
		require.ErrorIs(t, err, domain.ErrForbidden)
		require.Empty(t, reviews.created)
	})

	t.Run("заказ другого заведения", func(t *testing.T) {
		reviews, orders := reviewTestSetup()
		orders.order.VenueID = "venue-2"
		svc := NewReviewService(reviews, orders)
		_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{Rating: 5})
		require.ErrorIs(t, err, domain.ErrValidation)
		require.Empty(t, reviews.created)
	})

	t.Run("заказ ещё не доставлен", func(t *testing.T) {
		for _, status := range []domain.OrderStatus{
			domain.OrderStatusPending, domain.OrderStatusAccepted,
			domain.OrderStatusPreparing, domain.OrderStatusReady,
			domain.OrderStatusDelivering, domain.OrderStatusCancelled,
		} {
			reviews, orders := reviewTestSetup()
			orders.order.Status = status
			svc := NewReviewService(reviews, orders)
			_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{Rating: 5})
			require.ErrorIs(t, err, domain.ErrValidation, "status %s", status)
		}
		require.Empty(t, (&stubReviewRepo{}).created)
	})

	t.Run("второй отзыв по тому же заказу", func(t *testing.T) {
		reviews, orders := reviewTestSetup()
		reviews.existErr = nil
		reviews.existing = &domain.Review{ID: "r-1", UserID: "user-1", OrderID: "order-1", Rating: 4}
		svc := NewReviewService(reviews, orders)
		_, err := svc.Create(context.Background(), "user-1", "venue-1", "order-1", domain.CreateReviewRequest{Rating: 5})
		require.ErrorIs(t, err, domain.ErrAlreadyExists)
		require.Empty(t, reviews.created)
	})
}

func TestReviewListPagination(t *testing.T) {
	reviews, orders := reviewTestSetup()
	svc := NewReviewService(reviews, orders)

	_, _, err := svc.ListByVenue(context.Background(), "venue-1", 0, 0)
	require.NoError(t, err)
	require.Equal(t, [][2]int{{domain.DefaultPage, domain.DefaultPerPage}}, reviews.gotList)
}
