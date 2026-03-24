package request

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
)

type createNotificationConstraints struct {
	Title   string `validate:"required,max=200"`
	Content string `validate:"required,max=5000"`
	Type    string `validate:"omitempty,oneof=info warning success error"`
}

type updateNotificationConstraints struct {
	Title   string `validate:"required,max=200"`
	Content string `validate:"required,max=5000"`
	Type    string `validate:"omitempty,oneof=info warning success error"`
}

func validNotificationType(s string) (domain.NotificationType, bool) {
	switch domain.NotificationType(s) {
	case domain.NotificationInfo, domain.NotificationWarning, domain.NotificationSuccess, domain.NotificationError:
		return domain.NotificationType(s), true
	default:
		return "", false
	}
}

func ValidateCreateNotificationRequest(req *openapi.CreateNotificationRequest, v validator.Validator) error {
	t := string(lo.FromPtrOr(req.Type, openapi.CreateNotificationRequestType("")))
	if t == "" {
		t = "info"
	}
	c := createNotificationConstraints{Title: req.Title, Content: req.Content, Type: t}
	return ValidateConstraints(v, &c)
}

func ValidateCreateUserNotificationRequest(req *openapi.CreateUserNotificationRequest, v validator.Validator) error {
	t := string(lo.FromPtrOr(req.Type, openapi.CreateUserNotificationRequestType("")))
	if t == "" {
		t = "info"
	}
	c := createNotificationConstraints{Title: req.Title, Content: req.Content, Type: t}
	return ValidateConstraints(v, &c)
}

func ValidateUpdateNotificationRequest(req *openapi.UpdateNotificationRequest, v validator.Validator) error {
	t := string(lo.FromPtrOr(req.Type, openapi.UpdateNotificationRequestType("")))
	if t == "" {
		t = "info"
	}
	c := updateNotificationConstraints{Title: req.Title, Content: req.Content, Type: t}
	return ValidateConstraints(v, &c)
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
