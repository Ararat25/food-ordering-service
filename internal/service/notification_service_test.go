package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func TestNotificationRegisterToken(t *testing.T) {
	tokens := &stubPushTokenRepo{}
	svc := NewNotificationService(&stubNotifRepo{}, tokens)

	token := &domain.PushToken{UserID: "user-1", Token: "dev-token", Platform: "android"}
	require.NoError(t, svc.RegisterPushToken(context.Background(), token))
	require.NotEmpty(t, token.ID)
	require.Len(t, tokens.registered, 1)
	require.Equal(t, "dev-token", tokens.registered[0].Token)
}

func TestNotificationListPagination(t *testing.T) {
	notifs := &stubNotifRepo{}
	svc := NewNotificationService(notifs, &stubPushTokenRepo{})

	_, _, err := svc.ListByUser(context.Background(), "user-1", -1, 500)
	require.NoError(t, err)
	require.Equal(t, [][2]int{{domain.DefaultPage, domain.MaxPerPage}}, notifs.gotList)
}

func TestNotificationMarkAsRead(t *testing.T) {
	notifs := &stubNotifRepo{}
	svc := NewNotificationService(notifs, &stubPushTokenRepo{})

	require.NoError(t, svc.MarkAsRead(context.Background(), "n-1"))
	require.Equal(t, []string{"n-1"}, notifs.read)
}
