package repository

import (
	"context"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"

	"avito-kitchen/internal/domain"
)

func TestPushTokenRegisterDoesNotTransferOwnership(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPushTokenRepo(db)
	token := &domain.PushToken{ID: "token-row-1", UserID: "user-2", Token: "device-token", Platform: "android"}
	mock.ExpectQuery(`INSERT INTO push_tokens.*ON CONFLICT \(token\) DO UPDATE SET platform = EXCLUDED.platform WHERE push_tokens.user_id = EXCLUDED.user_id RETURNING id`).
		WithArgs(token.ID, token.UserID, token.Token, token.Platform).
		WillReturnRows(sqlmock.NewRows([]string{"id"}))

	require.ErrorIs(t, repo.Register(context.Background(), token), domain.ErrAlreadyExists)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPushTokenRegisterAllowsSameOwnerUpsert(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPushTokenRepo(db)
	token := &domain.PushToken{ID: "token-row-1", UserID: "user-1", Token: "device-token", Platform: "ios"}
	mock.ExpectQuery(`INSERT INTO push_tokens.*ON CONFLICT \(token\) DO UPDATE SET platform = EXCLUDED.platform WHERE push_tokens.user_id = EXCLUDED.user_id RETURNING id`).
		WithArgs(token.ID, token.UserID, token.Token, token.Platform).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(token.ID))

	require.NoError(t, repo.Register(context.Background(), token))
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestPushTokenDeleteIsScopedToOwner(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewPushTokenRepo(db)
	mock.ExpectExec(`DELETE FROM push_tokens WHERE token = \$1 AND user_id = \$2`).
		WithArgs("device-token", "user-1").
		WillReturnResult(sqlmock.NewResult(0, 0))

	require.ErrorIs(t, repo.DeleteForUser(context.Background(), "user-1", "device-token"), domain.ErrNotFound)
	require.NoError(t, mock.ExpectationsWereMet())
}
