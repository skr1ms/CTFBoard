package notification

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
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
	CountGlobal(ctx context.Context, sinceCreatedAt *time.Time) (int, error)
	Update(ctx context.Context, notif *domain.Notification) error
	Delete(ctx context.Context, ID uuid.UUID) error
	CreateUserNotification(ctx context.Context, userNotif *domain.UserNotification) error
	GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.UserNotification, error)
	GetUserNotificationByID(ctx context.Context, ID, userID uuid.UUID) (*domain.UserNotification, error)
	MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error
	CountUnread(ctx context.Context, userID uuid.UUID) (int, error)
}

type TeamReader interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.Team, error)
}

type NotificationUserReader interface {
	GetByID(ctx context.Context, ID uuid.UUID) (*domain.User, error)
	GetByTeamID(ctx context.Context, teamID uuid.UUID) ([]*domain.User, error)
}

type TransactionManager interface {
	Run(ctx context.Context, fn func(context.Context) error) error
}

type NotificationUseCase struct {
	deps NotificationDeps
}

type NotificationDeps struct {
	NotifRepo   NotificationRepository
	TeamRepo    TeamReader
	UserRepo    NotificationUserReader
	TM          TransactionManager
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

func (uc *NotificationUseCase) CreateGlobal(ctx context.Context, params usecase.NotificationCreateGlobalParams) (*domain.Notification, error) {
	if err := validateNotification(params.Title, params.Content, params.Type); err != nil {
		return nil, err
	}

	notif := &domain.Notification{
		ID:        uuid.New(),
		Title:     params.Title,
		Content:   params.Content,
		Type:      params.Type,
		IsPinned:  params.IsPinned,
		IsGlobal:  true,
		CreatedAt: time.Now(),
	}

	err := uc.deps.NotifRepo.Create(ctx, notif)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreateGlobal - NotificationRepo.Create: %w", err)
	}

	if uc.deps.Broadcaster != nil {
		txctx.AfterCommitOrNow(ctx, func(context.Context) {
			uc.deps.Broadcaster.NotifyNotification(notif.Title, string(notif.Type))
		})
	}

	return notif, nil
}

func (uc *NotificationUseCase) CreatePersonal(ctx context.Context, params usecase.NotificationCreatePersonalParams) (*domain.UserNotification, error) {
	if err := validateNotification(params.Title, params.Content, params.Type); err != nil {
		return nil, err
	}

	if uc.deps.UserRepo == nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal: dependencies not configured")
	}

	recipient, err := uc.deps.UserRepo.GetByID(ctx, params.UserID)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal - UserRepo.GetByID: %w", err)
	}

	if recipient == nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal - UserRepo.GetByID: %w", apperr.ErrUserNotFound)
	}

	userNotif := newUserNotification(params.UserID, params.Title, params.Content, params.Type)

	err = uc.deps.NotifRepo.CreateUserNotification(ctx, userNotif)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreatePersonal - NotificationRepo.CreateUserNotification: %w", err)
	}

	return userNotif, nil
}

func (uc *NotificationUseCase) CreateTeam(ctx context.Context, params usecase.NotificationCreateTeamParams) (*usecase.NotificationDeliveryResult, error) {
	if err := validateNotification(params.Title, params.Content, params.Type); err != nil {
		return nil, err
	}

	result := &usecase.NotificationDeliveryResult{
		TargetType: "team",
		TargetID:   params.TeamID,
	}

	if uc.deps.TeamRepo == nil || uc.deps.UserRepo == nil || uc.deps.TM == nil {
		return nil, fmt.Errorf("NotificationUseCase - CreateTeam: dependencies not configured")
	}

	if err := uc.deps.TM.Run(ctx, func(txCtx context.Context) error {
		team, err := uc.deps.TeamRepo.GetByID(txCtx, params.TeamID)
		if err != nil {
			return fmt.Errorf("TeamRepo.GetByID: %w", err)
		}

		if team.IsBanned {
			return apperr.ErrTeamBanned
		}

		members, err := uc.deps.UserRepo.GetByTeamID(txCtx, params.TeamID)
		if err != nil {
			return fmt.Errorf("UserRepo.GetByTeamID: %w", err)
		}

		for _, member := range members {
			if member == nil || member.IsBanned {
				continue
			}

			userNotif := newUserNotification(member.ID, params.Title, params.Content, params.Type)
			if err := uc.deps.NotifRepo.CreateUserNotification(txCtx, userNotif); err != nil {
				return fmt.Errorf("NotificationRepo.CreateUserNotification: %w", err)
			}

			result.CreatedCount++
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("NotificationUseCase - CreateTeam - TransactionManager.Run: %w", err)
	}

	return result, nil
}

func (uc *NotificationUseCase) GetGlobal(ctx context.Context, page, perPage int) ([]*domain.Notification, error) {
	offset := (page - 1) * perPage

	notifs, err := uc.deps.NotifRepo.GetAll(ctx, perPage, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - GetGlobal - NotificationRepo.GetAll: %w", err)
	}

	return notifs, nil
}

func (uc *NotificationUseCase) CountGlobal(ctx context.Context, sinceCreatedAt *time.Time) (int, error) {
	count, err := uc.deps.NotifRepo.CountGlobal(ctx, sinceCreatedAt)
	if err != nil {
		return 0, fmt.Errorf("NotificationUseCase - CountGlobal - NotificationRepo.CountGlobal: %w", err)
	}

	return count, nil
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

func validateNotification(title, content string, notificationType domain.NotificationType) error {
	if title == "" || content == "" {
		return apperr.ErrNotificationTitleContentRequired
	}

	if !notificationType.IsValid() {
		return apperr.NewValidationErrorf("invalid notification type %q", notificationType)
	}

	return nil
}

func newUserNotification(userID uuid.UUID, title, content string, notificationType domain.NotificationType) *domain.UserNotification {
	return &domain.UserNotification{
		ID:             uuid.New(),
		UserID:         userID,
		NotificationID: nil,
		Title:          title,
		Content:        content,
		Type:           notificationType,
		IsRead:         false,
		CreatedAt:      time.Now(),
	}
}

func (uc *NotificationUseCase) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := uc.deps.NotifRepo.CountUnread(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("NotificationUseCase - CountUnread - NotificationRepo.CountUnread: %w", err)
	}

	return count, nil
}

func (uc *NotificationUseCase) Update(ctx context.Context, params usecase.NotificationUpdateParams) (*domain.Notification, error) {
	if !params.Type.IsValid() {
		return nil, apperr.NewValidationErrorf("invalid notification type %q", params.Type)
	}

	notif, err := uc.deps.NotifRepo.GetByID(ctx, params.ID)
	if err != nil {
		return nil, fmt.Errorf("NotificationUseCase - Update - NotificationRepo.GetByID: %w", err)
	}

	notif.Title = params.Title
	notif.Content = params.Content
	notif.Type = params.Type

	notif.IsPinned = params.IsPinned
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
