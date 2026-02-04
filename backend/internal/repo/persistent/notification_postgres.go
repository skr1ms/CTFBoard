package persistent

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent/sqlc"
)

type NotificationRepo struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewNotificationRepo(pool *pgxpool.Pool) *NotificationRepo {
	return &NotificationRepo{
		pool: pool,
		q:    sqlc.New(pool),
	}
}

func (r *NotificationRepo) Create(ctx context.Context, notif *entity.Notification) error {
	if notif.ID == uuid.Nil {
		notif.ID = uuid.New()
	}
	if notif.CreatedAt.IsZero() {
		notif.CreatedAt = time.Now()
	}
	typeStr := string(notif.Type)
	row, err := r.q.CreateNotification(ctx, sqlc.CreateNotificationParams{
		ID:        notif.ID,
		Title:     notif.Title,
		Content:   notif.Content,
		Type:      &typeStr,
		IsPinned:  &notif.IsPinned,
		IsGlobal:  &notif.IsGlobal,
		CreatedAt: &notif.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("NotificationRepo - Create: %w", err)
	}
	notif.CreatedAt = ptrTimeToTime(row.CreatedAt)
	return nil
}

func (r *NotificationRepo) GetByID(ctx context.Context, id uuid.UUID) (*entity.Notification, error) {
	row, err := r.q.GetNotificationByID(ctx, id)
	if err != nil {
		if isNoRows(err) {
			return nil, entityError.ErrNotificationNotFound
		}
		return nil, fmt.Errorf("NotificationRepo - GetByID: %w", err)
	}
	return &entity.Notification{
		ID:        row.ID,
		Title:     row.Title,
		Content:   row.Content,
		Type:      entity.NotificationType(ptrStrToStr(row.Type)),
		IsPinned:  boolPtrToBool(row.IsPinned),
		IsGlobal:  boolPtrToBool(row.IsGlobal),
		CreatedAt: ptrTimeToTime(row.CreatedAt),
	}, nil
}

func (r *NotificationRepo) GetAll(ctx context.Context, limit, offset int) ([]*entity.Notification, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetAll limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetAll offset: %w", err)
	}
	rows, err := r.q.GetAllNotifications(ctx, sqlc.GetAllNotificationsParams{
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetAll: %w", err)
	}
	out := make([]*entity.Notification, len(rows))
	for i, row := range rows {
		out[i] = &entity.Notification{
			ID:        row.ID,
			Title:     row.Title,
			Content:   row.Content,
			Type:      entity.NotificationType(ptrStrToStr(row.Type)),
			IsPinned:  boolPtrToBool(row.IsPinned),
			IsGlobal:  boolPtrToBool(row.IsGlobal),
			CreatedAt: ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *NotificationRepo) Update(ctx context.Context, notif *entity.Notification) error {
	typeStr := string(notif.Type)
	return r.q.UpdateNotification(ctx, sqlc.UpdateNotificationParams{
		ID:       notif.ID,
		Title:    notif.Title,
		Content:  notif.Content,
		Type:     &typeStr,
		IsPinned: &notif.IsPinned,
	})
}

func (r *NotificationRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return r.q.DeleteNotification(ctx, id)
}

func (r *NotificationRepo) CreateUserNotification(ctx context.Context, userNotif *entity.UserNotification) error {
	if userNotif.ID == uuid.Nil {
		userNotif.ID = uuid.New()
	}
	if userNotif.CreatedAt.IsZero() {
		userNotif.CreatedAt = time.Now()
	}
	typeStr := string(userNotif.Type)
	isRead := userNotif.IsRead
	row, err := r.q.CreateUserNotification(ctx, sqlc.CreateUserNotificationParams{
		ID:             userNotif.ID,
		UserID:         userNotif.UserID,
		NotificationID: userNotif.NotificationID,
		Title:          strPtrOrNil(userNotif.Title),
		Content:        strPtrOrNil(userNotif.Content),
		Type:           &typeStr,
		IsRead:         &isRead,
		CreatedAt:      &userNotif.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("NotificationRepo - CreateUserNotification: %w", err)
	}
	userNotif.CreatedAt = ptrTimeToTime(row.CreatedAt)
	return nil
}

func (r *NotificationRepo) GetUserNotifications(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*entity.UserNotification, error) {
	limit32, err := intToInt32Safe(limit)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetUserNotifications limit: %w", err)
	}
	offset32, err := intToInt32Safe(offset)
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetUserNotifications offset: %w", err)
	}
	rows, err := r.q.GetUserNotifications(ctx, sqlc.GetUserNotificationsParams{
		UserID: userID,
		Limit:  limit32,
		Offset: offset32,
	})
	if err != nil {
		return nil, fmt.Errorf("NotificationRepo - GetUserNotifications: %w", err)
	}
	out := make([]*entity.UserNotification, len(rows))
	for i, row := range rows {
		out[i] = &entity.UserNotification{
			ID:             row.ID,
			UserID:         row.UserID,
			NotificationID: row.NotificationID,
			Title:          ptrStrToStr(row.Title),
			Content:        ptrStrToStr(row.Content),
			Type:           entity.NotificationType(ptrStrToStr(row.Type)),
			IsRead:         boolPtrToBool(row.IsRead),
			CreatedAt:      ptrTimeToTime(row.CreatedAt),
		}
	}
	return out, nil
}

func (r *NotificationRepo) MarkAsRead(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.MarkUserNotificationAsRead(ctx, sqlc.MarkUserNotificationAsReadParams{
		ID:     id,
		UserID: userID,
	})
}

func (r *NotificationRepo) CountUnread(ctx context.Context, userID uuid.UUID) (int, error) {
	count, err := r.q.CountUnreadUserNotifications(ctx, userID)
	if err != nil {
		return 0, fmt.Errorf("NotificationRepo - CountUnread: %w", err)
	}
	return int(count), nil
}

func (r *NotificationRepo) DeleteUserNotification(ctx context.Context, id, userID uuid.UUID) error {
	return r.q.DeleteUserNotification(ctx, sqlc.DeleteUserNotificationParams{
		ID:     id,
		UserID: userID,
	})
}
