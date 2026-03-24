package response

import (
	"time"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromNotification(n *domain.Notification) openapi.NotificationResponse {
	return openapi.NotificationResponse{
		ID:        httputil.Ptr(n.ID.String()),
		Title:     httputil.Ptr(n.Title),
		Content:   httputil.Ptr(n.Content),
		Type:      httputil.Ptr(string(n.Type)),
		IsPinned:  httputil.Ptr(n.IsPinned),
		CreatedAt: httputil.Ptr(n.CreatedAt.Format(time.RFC3339)),
	}
}

func FromNotificationList(ns []*domain.Notification) []openapi.NotificationResponse {
	return lo.Map(ns, func(n *domain.Notification, _ int) openapi.NotificationResponse { return FromNotification(n) })
}

func FromUserNotification(un *domain.UserNotification) openapi.UserNotificationResponse {
	return openapi.UserNotificationResponse{
		ID:        httputil.Ptr(un.ID.String()),
		Title:     httputil.Ptr(un.Title),
		Content:   httputil.Ptr(un.Content),
		Type:      httputil.Ptr(string(un.Type)),
		IsRead:    httputil.Ptr(un.IsRead),
		CreatedAt: httputil.Ptr(un.CreatedAt.Format(time.RFC3339)),
	}
}

func FromUserNotificationList(uns []*domain.UserNotification) []openapi.UserNotificationResponse {
	return lo.Map(uns, func(un *domain.UserNotification, _ int) openapi.UserNotificationResponse {
		return FromUserNotification(un)
	})
}
