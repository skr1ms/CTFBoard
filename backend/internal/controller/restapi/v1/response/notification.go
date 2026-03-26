package response

import (
	"time"

	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromNotification(n *domain.Notification) openapi.NotificationResponse {
	return openapi.NotificationResponse{
		ID:        new(n.ID.String()),
		Title:     new(n.Title),
		Content:   new(n.Content),
		Type:      new(string(n.Type)),
		IsPinned:  new(n.IsPinned),
		CreatedAt: new(n.CreatedAt.Format(time.RFC3339)),
	}
}

func FromNotificationList(ns []*domain.Notification) []openapi.NotificationResponse {
	return lo.Map(ns, func(n *domain.Notification, _ int) openapi.NotificationResponse { return FromNotification(n) })
}

func FromUserNotification(un *domain.UserNotification) openapi.UserNotificationResponse {
	return openapi.UserNotificationResponse{
		ID:        new(un.ID.String()),
		Title:     new(un.Title),
		Content:   new(un.Content),
		Type:      new(string(un.Type)),
		IsRead:    new(un.IsRead),
		CreatedAt: new(un.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserNotificationList(uns []*domain.UserNotification) []openapi.UserNotificationResponse {
	return lo.Map(uns, func(un *domain.UserNotification, _ int) openapi.UserNotificationResponse {
		return FromUserNotification(un)
	})
}
