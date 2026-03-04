package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func CreateNotificationRequestToParams(req *openapi.CreateNotificationRequest) (title, content string, notifType entity.NotificationType, isPinned bool) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}
	return req.Title, req.Content, notifType, derefOr(req.IsPinned, false)
}

func CreateUserNotificationRequestToParams(req *openapi.CreateUserNotificationRequest) (title, content string, notifType entity.NotificationType) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}
	return req.Title, req.Content, notifType
}

func UpdateNotificationRequestToParams(req *openapi.UpdateNotificationRequest) (title, content string, notifType entity.NotificationType, isPinned bool) {
	notifType = entity.NotificationInfo
	if req.Type != nil {
		notifType = entity.NotificationType(*req.Type)
	}
	return req.Title, req.Content, notifType, derefOr(req.IsPinned, false)
}
