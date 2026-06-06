package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// NotificationBroadcaster broadcasts a notification to connected WebSocket clients. Optional.
type NotificationBroadcaster interface {
	NotifyNotification(message, level string)
}

type NotificationRepository interface {
	Create(ctx context.Context, notif *domain.Notification) error
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Notification, error)
	GetAll(ctx context.Context, limit, offset int) ([]*domain.Notification, error)
	Update(ctx context.Context, notif *domain.Notification) error
	Delete(ctx context.Context, ID uuid.UUID) error
	CreateUserNotification(ctx context.Context, userNotif *domain.UserNotification) error
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.UserNotification, error)
	GetUserNotificationByID(ctx context.Context, ID, userID uuid.UUID) (*domain.UserNotification, error)
	MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

type NotificationUseCase struct {
	deps NotificationDeps
}

type NotificationDeps struct {
	NotifRepo   NotificationRepository
	Broadcaster NotificationBroadcaster
	Logger      logkit.Logger
}

var _ usecase.NotificationUseCase = (*NotificationUseCase)(nil)

func NewNotificationUseCase(deps NotificationDeps) *NotificationUseCase {
	if deps.Logger == nil {
		deps.Logger = logkit.Noop()
	}

	return &NotificationUseCase{deps: deps}
}

func (uc *NotificationUseCase) CreateGlobal(ctx context.Context, title, content string, notifType domain.NotificationType, isPinned bool) (*domain.Notification, error) {
	if title == "" || content == "" {
		return nil, apperr.ErrNotificationTitleContentRequired
	}

	if !notifType.IsValid() {
		return nil, apperr.NewValidationErrorf("invalid notification type %q", notifType)
	}

	notif := &domain.Notification{
		ID:        uuid.New(),
		Title:     title,
		Content:   content,
		Type:      notifType,
		IsPinned:  isPinned,
		IsGlobal:  true,
		CreatedAt: time.Now(),
	}

	err := uc.deps.NotifRepo.Create(ctx, notif)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreateGlobal - NotificationRepo.Create: %w", err)
	}

	if uc.deps.Broadcaster != nil {
		uc.deps.Broadcaster.NotifyNotification(notif.Title, string(notif.Type))
	}

	return notif, nil
}

func (uc *NotificationUseCase) CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType domain.NotificationType) (*domain.UserNotification, error) {
	if title == "" || content == "" {
		return nil, apperr.ErrNotificationTitleContentRequired
	}

	if !notifType.IsValid() {
		return nil, apperr.NewValidationErrorf("invalid notification type %q", notifType)
	}

	userNotif := &domain.UserNotification{
		ID:             uuid.New(),
		UserID:         userID,
		NotificationID: nil,
		Title:          title,
		Content:        content,
		Type:           notifType,
		IsRead:         false,
		CreatedAt:      time.Now(),
	}

	err := uc.deps.NotifRepo.CreateUserNotification(ctx, userNotif)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal - NotificationRepo.CreateUserNotification: %w", err)
	}

	return userNotif, nil
}

func (uc *NotificationUseCase) GetGlobal(ctx context.Context, page, perPage int) ([]*domain.Notification, error) {
	offset := (page - 1) * perPage

	notifs, err := uc.deps.NotifRepo.GetAll(ctx, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - GetGlobal - NotificationRepo.GetAll: %w", err)
	}

	return notifs, nil
}

func (uc *NotificationUseCase) GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*domain.UserNotification, error) {
	offset := (page - 1) * perPage

	userNotifs, err := uc.deps.NotifRepo.GetUserNotifications(ctx, userID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - GetUserNotifications - NotificationRepo.GetUserNotifications: %w", err)
	}

	return userNotifs, nil
}

func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error {
	if _, err := uc.deps.NotifRepo.GetUserNotificationByID(ctx, ID, userID); err != nil {
		return fmt.Errorf("NotificationUseCase - MarkAsRead - GetUserNotificationByID: %w", err)
	}

	err := uc.deps.NotifRepo.MarkAsRead(ctx, ID, userID)
	if err != nil {
		return fmt.Errorf("NotificationUseCase - MarkAsRead - NotificationRepo.MarkAsRead: %w", err)
	}

	return nil
}

func (uc *NotificationUseCase) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := uc.deps.NotifRepo.CountUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("NotificationUseCase - CountUnread - NotificationRepo.CountUnread: %w", err)
	}

	return count, nil
}

func (uc *NotificationUseCase) Update(ctx context.Context, ID uuid.UUID, title, content string, notifType domain.NotificationType, isPinned bool) (*domain.Notification, error) {
	if !notifType.IsValid() {
		return nil, apperr.NewValidationErrorf("invalid notification type %q", notifType)
	}

	notif, err := uc.deps.NotifRepo.GetByID(ctx, ID)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - Update - NotificationRepo.GetByID: %w", err)
	}

	notif.Title = title
	notif.Content = content
	notif.Type = notifType

	notif.IsPinned = isPinned
	if err := uc.deps.NotifRepo.Update(ctx, notif); err != nil {
		return nil, fmt.Errorf("NotificationUseCase - Update - NotificationRepo.Update: %w", err)
	}

	return notif, nil
}

func (uc *NotificationUseCase) Delete(ctx context.Context, ID uuid.UUID) error {
	err := uc.deps.NotifRepo.Delete(ctx, ID)
	if err != nil {
		return fmt.Errorf("NotificationUseCase - Delete - NotificationRepo.Delete: %w", err)
	}

	return nil
}
