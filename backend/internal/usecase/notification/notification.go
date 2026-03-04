package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
)

// NotificationBroadcaster broadcasts a notification to connected WebSocket clients. Optional.
type NotificationBroadcaster interface {
	NotifyNotification(message, level string)
}

type NotificationUseCase struct {
	deps NotificationDeps
}

type NotificationDeps struct {
	NotifRepo   repo.NotificationRepository
	Broadcaster NotificationBroadcaster
}

var _ usecase.NotificationUseCase = (*NotificationUseCase)(nil)

func NewNotificationUseCase(deps NotificationDeps) *NotificationUseCase {
	return &NotificationUseCase{deps: deps}
}

func (uc *NotificationUseCase) CreateGlobal(ctx context.Context, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error) {
	if title == "" || content == "" {
		return nil, httperr.ErrNotificationTitleContentRequired
	}
	notif := &entity.Notification{
		ID:        uuid.New(),
		Title:     title,
		Content:   content,
		Type:      notifType,
		IsPinned:  isPinned,
		IsGlobal:  true,
		CreatedAt: time.Now(),
	}
	if err := uc.deps.NotifRepo.Create(ctx, notif); err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreateGlobal - NotificationRepo.Create: %w", err)
	}
	if uc.deps.Broadcaster != nil {
		uc.deps.Broadcaster.NotifyNotification(notif.Title, string(notif.Type))
	}
	return notif, nil
}

func (uc *NotificationUseCase) CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType entity.NotificationType) (*entity.UserNotification, error) {
	if title == "" || content == "" {
		return nil, httperr.ErrNotificationTitleContentRequired
	}
	userNotif := &entity.UserNotification{
		ID:             uuid.New(),
		UserID:         userID,
		NotificationID: nil,
		Title:          title,
		Content:        content,
		Type:           notifType,
		IsRead:         false,
		CreatedAt:      time.Now(),
	}
	if err := uc.deps.NotifRepo.CreateUserNotification(ctx, userNotif); err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal - NotificationRepo.CreateUserNotification: %w", err)
	}
	return userNotif, nil
}

func (uc *NotificationUseCase) GetGlobal(ctx context.Context, page, perPage int) ([]*entity.Notification, error) {
	offset := (page - 1) * perPage
	notifs, err := uc.deps.NotifRepo.GetAll(ctx, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - GetGlobal - NotificationRepo.GetAll: %w", err)
	}
	return notifs, nil
}

func (uc *NotificationUseCase) GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.UserNotification, error) {
	offset := (page - 1) * perPage
	userNotifs, err := uc.deps.NotifRepo.GetUserNotifications(ctx, userID, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - GetUserNotifications - NotificationRepo.GetUserNotifications: %w", err)
	}
	return userNotifs, nil
}

func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error {
	if err := uc.deps.NotifRepo.MarkAsRead(ctx, ID, userID); err != nil {
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

func (uc *NotificationUseCase) Update(ctx context.Context, ID uuid.UUID, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error) {
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
	if err := uc.deps.NotifRepo.Delete(ctx, ID); err != nil {
		return fmt.Errorf("NotificationUseCase - Delete - NotificationRepo.Delete: %w", err)
	}
	return nil
}
