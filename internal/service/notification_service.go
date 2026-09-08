package service

import (
	"context"

	"github.com/google/uuid"

	"avito-kitchen/internal/domain"
)

// NotificationService - логика уведомлений.
type NotificationService struct {
	notifRepo domain.NotificationRepository // уведомления в базе
	tokenRepo domain.PushTokenRepository    // пуш-токены в базе
}

// NewNotificationService - создаёт сервис уведомлений.
func NewNotificationService(notifRepo domain.NotificationRepository, tokenRepo domain.PushTokenRepository) *NotificationService {
	return &NotificationService{notifRepo: notifRepo, tokenRepo: tokenRepo}
}

// Create - создаёт уведомление.
func (s *NotificationService) Create(ctx context.Context, n *domain.Notification) error {
	return s.notifRepo.Create(ctx, n)
}

// ListByUser - возвращает уведомления пользователя постранично.
func (s *NotificationService) ListByUser(ctx context.Context, userID string, page, perPage int) ([]domain.Notification, int, error) {
	page, perPage = domain.NormalizePagination(page, perPage)
	return s.notifRepo.ListByUser(ctx, userID, page, perPage)
}

// MarkAsRead - помечает уведомление прочитанным.
func (s *NotificationService) MarkAsRead(ctx context.Context, id string) error {
	return s.notifRepo.MarkAsRead(ctx, id)
}

// RegisterPushToken - сохраняет пуш-токен, без id придумывает новый.
func (s *NotificationService) RegisterPushToken(ctx context.Context, token *domain.PushToken) error {
	if token.ID == "" {
		token.ID = uuid.New().String()
	}
	return s.tokenRepo.Register(ctx, token)
}

// UnregisterPushToken - отзывает токен устройства только у его владельца.
func (s *NotificationService) UnregisterPushToken(ctx context.Context, userID, token string) error {
	return s.tokenRepo.DeleteForUser(ctx, userID, token)
}
