package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

const (
	maxNotificationTitleLength   = 200
	maxNotificationContentLength = 5000
)

func validNotificationType(s string) (entity.NotificationType, bool) {
	switch entity.NotificationType(s) {
	case entity.NotificationInfo, entity.NotificationWarning, entity.NotificationSuccess, entity.NotificationError:
		return entity.NotificationType(s), true
	default:
		return "", false
	}
}

func CreateNotificationRequestToParams(req *openapi.CreateNotificationRequest) (title, content string, notifType entity.NotificationType, isPinned bool, err error) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		} else {
			return "", "", "", false, helper.NewValidationErrorf("invalid notification type: must be one of info, warning, success, error")
		}
	}
	if len(req.Title) > maxNotificationTitleLength {
		return "", "", "", false, helper.NewValidationErrorf("title must be at most %d characters", maxNotificationTitleLength)
	}
	if len(req.Content) > maxNotificationContentLength {
		return "", "", "", false, helper.NewValidationErrorf("content must be at most %d characters", maxNotificationContentLength)
	}
	return req.Title, req.Content, notifType, derefOr(req.IsPinned, false), nil
}

func CreateUserNotificationRequestToParams(req *openapi.CreateUserNotificationRequest) (title, content string, notifType entity.NotificationType, err error) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		} else {
			return "", "", "", helper.NewValidationErrorf("invalid notification type: must be one of info, warning, success, error")
		}
	}
	if len(req.Title) > maxNotificationTitleLength {
		return "", "", "", helper.NewValidationErrorf("title must be at most %d characters", maxNotificationTitleLength)
	}
	if len(req.Content) > maxNotificationContentLength {
		return "", "", "", helper.NewValidationErrorf("content must be at most %d characters", maxNotificationContentLength)
	}
	return req.Title, req.Content, notifType, nil
}

func UpdateNotificationRequestToParams(req *openapi.UpdateNotificationRequest) (title, content string, notifType entity.NotificationType, isPinned bool, err error) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		} else {
			return "", "", "", false, helper.NewValidationErrorf("invalid notification type: must be one of info, warning, success, error")
		}
	}
	if len(req.Title) > maxNotificationTitleLength {
		return "", "", "", false, helper.NewValidationErrorf("title must be at most %d characters", maxNotificationTitleLength)
	}
	if len(req.Content) > maxNotificationContentLength {
		return "", "", "", false, helper.NewValidationErrorf("content must be at most %d characters", maxNotificationContentLength)
	}
	return req.Title, req.Content, notifType, derefOr(req.IsPinned, false), nil
}
