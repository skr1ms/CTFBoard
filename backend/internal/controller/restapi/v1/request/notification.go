package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func validNotificationType(s string) (domain.NotificationType, bool) {
	switch domain.NotificationType(s) {
	case domain.NotificationInfo, domain.NotificationWarning, domain.NotificationSuccess, domain.NotificationError:
		return domain.NotificationType(s), true
	default:
		return "", false
	}
}

func CreateNotificationRequestToParams(req *openapi.CreateNotificationRequest) (title, content string, notifType domain.NotificationType, isPinned bool, err error) {
	notifType = domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return req.Title, req.Content, notifType, lo.FromPtrOr(req.IsPinned, false), nil
}

func CreateUserNotificationRequestToParams(req *openapi.CreateUserNotificationRequest) (title, content string, notifType domain.NotificationType, err error) {
	notifType = domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return req.Title, req.Content, notifType, nil
}

func UpdateNotificationRequestToParams(req *openapi.UpdateNotificationRequest) (title, content string, notifType domain.NotificationType, isPinned bool, err error) {
	notifType = domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return req.Title, req.Content, notifType, lo.FromPtrOr(req.IsPinned, false), nil
}
