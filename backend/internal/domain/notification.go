package domain

import (
	"time"

	"github.com/google/uuid"
)

// NotificationType is a string-backed enum for the severity/style of a notification.
type NotificationType string

const (
	// NotificationInfo represents an informational notification.
	NotificationInfo NotificationType = "info"
	// NotificationWarning represents a warning notification.
	NotificationWarning NotificationType = "warning"
	// NotificationSuccess represents a success notification.
	NotificationSuccess NotificationType = "success"
	// NotificationError represents an error notification.
	NotificationError NotificationType = "error"
)

// IsValid returns true if t is one of the four recognized notification types.
func (t NotificationType) IsValid() bool {
	switch t {
	case NotificationInfo, NotificationWarning, NotificationSuccess, NotificationError:
		return true
	default:
		return false
	}
}

// Notification is a broadcast message created by admins, optionally pinned or sent globally.
type Notification struct {
	ID        uuid.UUID        `json:"id"`
	Title     string           `json:"title"`
	Content   string           `json:"content"`
	Type      NotificationType `json:"type"`
	IsPinned  bool             `json:"is_pinned"`
	IsGlobal  bool             `json:"is_global"`
	CreatedAt time.Time        `json:"created_at"`
}

// UserNotification links a notification to a specific user and tracks whether it has been read.
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
