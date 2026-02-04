package entity

import (
	"time"

	"github.com/google/uuid"
)

type NotificationType string

const (
	NotificationInfo    NotificationType = "info"
	NotificationWarning NotificationType = "warning"
	NotificationSuccess NotificationType = "success"
	NotificationError   NotificationType = "error"
)

type Notification struct {
	ID        uuid.UUID        `json:"id"`
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Type      NotificationType `json:"type"`
	IsPinned  bool             `json:"is_pinned"`
	IsGlobal  bool             `json:"is_global"`
	CreatedAt time.Time        `json:"created_at"`
}

type UserNotification struct {
	ID             uuid.UUID        `json:"id"`
	UserID         uuid.UUID        `json:"user_id"`
	NotificationID *uuid.UUID       `json:"notification_id,omitempty"`
	Title          string           `json:"title"`
	Content        string           `json:"content"`
	Type           NotificationType `json:"type"`
	IsRead         bool             `json:"is_read"`
	CreatedAt      time.Time        `json:"created_at"`
}
