package response

import (
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
)

func FromNotification(n *entity.Notification) openapi.ResponseNotificationResponse {
	return openapi.ResponseNotificationResponse{
		ID:        ptr(n.ID.String()),
		Title:     ptr(n.Title),
		Content:   ptr(n.Content),
		Type:      ptr(string(n.Type)),
		IsPinned:  ptr(n.IsPinned),
		CreatedAt: ptr(n.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserNotification(un *entity.UserNotification) openapi.ResponseUserNotificationResponse {
	return openapi.ResponseUserNotificationResponse{
		ID:        ptr(un.ID.String()),
		Title:     ptr(un.Title),
		Content:   ptr(un.Content),
		Type:      ptr(string(un.Type)),
		IsRead:    ptr(un.IsRead),
		CreatedAt: ptr(un.CreatedAt.Format(time.RFC3339)),
	}
}
