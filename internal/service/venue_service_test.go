package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func TestVenueListPagination(t *testing.T) {
	repo := &stubVenueRepo{listOut: []domain.Venue{{ID: "v-1", Name: "Итальянец"}}, listN: 1}
	svc := NewVenueService(repo)

	venues, total, err := svc.List(context.Background(), domain.VenueFilters{})
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, venues, 1)
	require.Len(t, repo.gotList, 1)
	require.Equal(t, domain.DefaultPage, repo.gotList[0].Page)
	require.Equal(t, domain.DefaultPerPage, repo.gotList[0].PerPage)
}

func TestVenueGetByID(t *testing.T) {
	t.Run("находят", func(t *testing.T) {
		repo := &stubVenueRepo{venue: &domain.Venue{ID: "v-1", Name: "Итальянец"}}
		svc := NewVenueService(repo)
		got, err := svc.GetByID(context.Background(), "v-1")
		require.NoError(t, err)
		require.Equal(t, "Итальянец", got.Name)
	})

	t.Run("нет такого - 404", func(t *testing.T) {
		repo := &stubVenueRepo{getErr: domain.ErrNotFound}
		svc := NewVenueService(repo)
		_, err := svc.GetByID(context.Background(), "nope")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestVenueGetByAPIKey(t *testing.T) {
	t.Run("ключ верный", func(t *testing.T) {
		repo := &stubVenueRepo{venue: &domain.Venue{ID: "v-1"}}
		svc := NewVenueService(repo)
		got, err := svc.GetByAPIKey(context.Background(), "vk_good")
		require.NoError(t, err)
		require.Equal(t, "v-1", got.ID)
	})

	t.Run("ключ левый", func(t *testing.T) {
		repo := &stubVenueRepo{keyErr: domain.ErrNotFound}
		svc := NewVenueService(repo)
		_, err := svc.GetByAPIKey(context.Background(), "vk_bad")
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}
