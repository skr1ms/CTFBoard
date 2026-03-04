package response

import (
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromNotification(n *entity.Notification) openapi.NotificationResponse {
	return openapi.NotificationResponse{
		ID:        ptr(n.ID.String()),
		Title:     ptr(n.Title),
		Content:   ptr(n.Content),
		Type:      ptr(string(n.Type)),
		IsPinned:  ptr(n.IsPinned),
		CreatedAt: ptr(n.CreatedAt.Format(time.RFC3339)),
	}
}

func FromNotificationList(ns []*entity.Notification) []openapi.NotificationResponse {
	res := make([]openapi.NotificationResponse, len(ns))
	for i, n := range ns {
		res[i] = FromNotification(n)
	}
	return res
}

func FromUserNotification(un *entity.UserNotification) openapi.UserNotificationResponse {
	return openapi.UserNotificationResponse{
		ID:        ptr(un.ID.String()),
		Title:     ptr(un.Title),
		Content:   ptr(un.Content),
		Type:      ptr(string(un.Type)),
		IsRead:    ptr(un.IsRead),
		CreatedAt: ptr(un.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserNotificationList(uns []*entity.UserNotification) []openapi.UserNotificationResponse {
	res := make([]openapi.UserNotificationResponse, len(uns))
	for i, un := range uns {
		res[i] = FromUserNotification(un)
	}
	return res
}
