package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/samber/lo"

	"github.com/wahrwelt-kit/go-pgkit/pgutil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo/persistent/sqlc"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type NotificationRepo struct {
	BaseRepo
}

var _ repo.NotificationRepository = (*NotificationRepo)(nil)

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{BaseRepo: BaseRepo{pool: pool}}
}

func (r *NotificationRepo) Create(ctx context.Context, notif *domain.Notification) error {
	EnsureID(&notif.ID)
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	typeStr := string(notif.Type)
	row, err := r.Q(ctx).CreateNotification(ctx, sqlc.CreateNotificationParams{
		ID:        notif.ID,
		Title:     notif.Title,
		Content:   notif.Content,
		Type:      &typeStr,
		IsPinned:  &notif.IsPinned,
		IsGlobal:  &notif.IsGlobal,
		CreatedAt: pgutil.TimeToTimestamptz(&notif.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("NotificationRepo - Create: %w", err)
	}
	notif.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	return nil
}

func (r *NotificationRepo) GetByID(ctx context.Context, ID uuid.UUID) (*domain.Notification, error) {
	row, err := r.Q(ctx).GetNotificationByID(ctx, ID)
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("NotificationRepo - GetByID: %w", err)
	}
	return &domain.Notification{
		ID:        row.ID,
		Title:     row.Title,
		Content:   row.Content,
		Type:      domain.NotificationType(lo.FromPtr(row.Type)),
		IsPinned:  lo.FromPtr(row.IsPinned),
		IsGlobal:  lo.FromPtr(row.IsGlobal),
		CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
	}, nil
}

func (r *NotificationRepo) GetAll(ctx context.Context, limit, offset int) ([]*domain.Notification, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetAll: %w", err)
	}
	rows, err := r.Q(ctx).GetAllNotifications(ctx, sqlc.GetAllNotificationsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetAll: %w", err)
	}
	out := make([]*domain.Notification, len(rows))
	for i, row := range rows {
		out[i] = &domain.Notification{
			ID:        row.ID,
			Title:     row.Title,
			Content:   row.Content,
			Type:      domain.NotificationType(lo.FromPtr(row.Type)),
			IsPinned:  lo.FromPtr(row.IsPinned),
			IsGlobal:  lo.FromPtr(row.IsGlobal),
			CreatedAt: pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *NotificationRepo) Update(ctx context.Context, notif *domain.Notification) error {
	typeStr := string(notif.Type)
	if err := r.Q(ctx).UpdateNotification(ctx, sqlc.UpdateNotificationParams{
		ID:       notif.ID,
		Title:    notif.Title,
		Content:  notif.Content,
		Type:     &typeStr,
		IsPinned: &notif.IsPinned,
	}); err != nil {
		if pgutil.IsNoRows(err) {
			return httperr.ErrNotificationNotFound
		}
		return fmt.Errorf("NotificationRepo - Update: %w", err)
	}
	return nil
}

func (r *NotificationRepo) Delete(ctx context.Context, ID uuid.UUID) error {
	if err := r.Q(ctx).DeleteNotification(ctx, ID); err != nil {
		return fmt.Errorf("NotificationRepo - Delete: %w", err)
	}
	return nil
}

func (r *NotificationRepo) CreateUserNotification(ctx context.Context, userNotif *domain.UserNotification) error {
	EnsureID(&userNotif.ID)
	if userNotif.CreatedAt.IsZero() {
		userNotif.CreatedAt = time.Now()
	}
	typeStr := string(userNotif.Type)
	isRead := userNotif.IsRead
	row, err := r.Q(ctx).CreateUserNotification(ctx, sqlc.CreateUserNotificationParams{
		ID:             userNotif.ID,
		UserID:         userNotif.UserID,
		NotificationID: userNotif.NotificationID,
		Title:          lo.EmptyableToPtr(userNotif.Title),
		Content:        lo.EmptyableToPtr(userNotif.Content),
		Type:           &typeStr,
		IsRead:         &isRead,
		CreatedAt:      pgutil.TimeToTimestamptz(&userNotif.CreatedAt),
	})
	if err != nil {
		return fmt.Errorf("NotificationRepo - CreateUserNotification: %w", err)
	}
	userNotif.CreatedAt = pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt))
	return nil
}

func (r *NotificationRepo) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*domain.UserNotification, error) {
	limit32, offset32, err := toLimitOffset(limit, offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetUserNotifications: %w", err)
	}
	rows, err := r.Q(ctx).GetUserNotifications(ctx, sqlc.GetUserNotificationsParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetUserNotifications: %w", err)
	}
	out := make([]*domain.UserNotification, len(rows))
	for i, row := range rows {
		out[i] = &domain.UserNotification{
			ID:             row.ID,
			UserID:         row.UserID,
			NotificationID: row.NotificationID,
			Title:          lo.FromPtr(row.Title),
			Content:        lo.FromPtr(row.Content),
			Type:           domain.NotificationType(lo.FromPtr(row.Type)),
			IsRead:         lo.FromPtr(row.IsRead),
			CreatedAt:      pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
		}
	}
	return out, nil
}

func (r *NotificationRepo) GetUserNotificationByID(ctx context.Context, ID, userID uuid.UUID) (*domain.UserNotification, error) {
	row, err := r.Q(ctx).GetUserNotificationByID(ctx, sqlc.GetUserNotificationByIDParams{
		ID:     ID,
		UserID: userID,
	})
	if err != nil {
		if pgutil.IsNoRows(err) {
			return nil, httperr.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("NotificationRepo - GetUserNotificationByID: %w", err)
	}
	return &domain.UserNotification{
		ID:             row.ID,
		UserID:         row.UserID,
		NotificationID: row.NotificationID,
		Title:          lo.FromPtr(row.Title),
		Content:        lo.FromPtr(row.Content),
		Type:           domain.NotificationType(lo.FromPtr(row.Type)),
		IsRead:         lo.FromPtr(row.IsRead),
		CreatedAt:      pgutil.PtrTimeToTime(pgutil.TimestamptzToTime(row.CreatedAt)),
	}, nil
}

func (r *NotificationRepo) MarkAsRead(ctx context.Context, ID, userID uuid.UUID) error {
	if err := r.Q(ctx).MarkUserNotificationAsRead(ctx, sqlc.MarkUserNotificationAsReadParams{
		ID:     ID,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("NotificationRepo - MarkAsRead: %w", err)
	}
	return nil
}

func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := r.Q(ctx).CountUnreadUserNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("NotificationRepo - CountUnread: %w", err)
	}
	return int(count), nil
}

func (r *NotificationRepo) DeleteUserNotification(ctx context.Context, ID, userID uuid.UUID) error {
	if err := r.Q(ctx).DeleteUserNotification(ctx, sqlc.DeleteUserNotificationParams{
		ID:     ID,
		UserID: userID,
	}); err != nil {
		return fmt.Errorf("NotificationRepo - DeleteUserNotification: %w", err)
	}
	return nil
}
