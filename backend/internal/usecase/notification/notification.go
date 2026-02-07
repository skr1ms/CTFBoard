package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
)

type NotificationUseCase struct {
	notifRepo repo.NotificationRepository
}

func NewNotificationUseCase(
	notifRepo repo.NotificationRepository,
) *NotificationUseCase {
	return &NotificationUseCase{notifRepo: notifRepo}
}

func (uc *NotificationUseCase) CreateGlobal(ctx context.Context, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error) {
	if title == "" || content == "" {
		return nil, fmt.Errorf("NotificationUseCase - CreateGlobal: title and content are required")
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
	if err := uc.notifRepo.Create(ctx, notif); err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - CreateGlobal")
	}
	return notif, nil
}

func (uc *NotificationUseCase) CreatePersonal(ctx context.Context, userID uuid.UUID, title, content string, notifType entity.NotificationType) (*entity.UserNotification, error) {
	if title == "" || content == "" {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal: title and content are required")
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
	if err := uc.notifRepo.CreateUserNotification(ctx, userNotif); err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - CreatePersonal")
	}
	return userNotif, nil
}

func (uc *NotificationUseCase) GetGlobal(ctx context.Context, page, perPage int) ([]*entity.Notification, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	notifs, err := uc.notifRepo.GetAll(ctx, perPage, offset)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - GetGlobal")
	}
	return notifs, nil
}

func (uc *NotificationUseCase) GetUserNotifications(ctx context.Context, userID uuid.UUID, page, perPage int) ([]*entity.UserNotification, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage
	userNotifs, err := uc.notifRepo.GetUserNotifications(ctx, userID, perPage, offset)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - GetUserNotifications")
	}
	return userNotifs, nil
}

func (uc *NotificationUseCase) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	if err := uc.notifRepo.MarkAsRead(ctx, id, userID); err != nil {
		return usecaseutil.Wrap(err, "NotificationUseCase - MarkAsRead")
	}
	return nil
}

func (uc *NotificationUseCase) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := uc.notifRepo.CountUnread(ctx, userID)
	if err != nil {
		return 0, usecaseutil.Wrap(err, "NotificationUseCase - CountUnread")
	}
	return count, nil
}

func (uc *NotificationUseCase) Update(ctx context.Context, id uuid.UUID, title, content string, notifType entity.NotificationType, isPinned bool) (*entity.Notification, error) {
	notif, err := uc.notifRepo.GetByID(ctx, id)
	if err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - Update - GetByID")
	}
	notif.Title = title
	notif.Content = content
	notif.Type = notifType
	notif.IsPinned = isPinned
	if err := uc.notifRepo.Update(ctx, notif); err != nil {
		return nil, usecaseutil.Wrap(err, "NotificationUseCase - Update")
	}
	return notif, nil
}

func (uc *NotificationUseCase) Delete(ctx context.Context, id uuid.UUID) error {
	if err := uc.notifRepo.Delete(ctx, id); err != nil {
		return usecaseutil.Wrap(err, "NotificationUseCase - Delete")
	}
	return nil
}
