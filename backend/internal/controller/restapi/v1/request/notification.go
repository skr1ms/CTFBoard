package request

import (
	"github.com/google/uuid"
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func validNotificationType(s string) (domain.NotificationType, bool) {
	switch domain.NotificationType(s) {
	case domain.NotificationInfo, domain.NotificationWarning, domain.NotificationSuccess, domain.NotificationError:
		return domain.NotificationType(s), true
	default:
		return "", false
	}
}

func CreateNotificationRequestToParams(req *openapi.CreateNotificationRequest) (usecase.NotificationCreateGlobalParams, error) {
	notifType := domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return usecase.NotificationCreateGlobalParams{
		Title:    req.Title,
		Content:  req.Content,
		Type:     notifType,
		IsPinned: lo.FromPtrOr(req.IsPinned, false),
	}, nil
}

func CreateUserNotificationRequestToParams(req *openapi.CreateUserNotificationRequest, userID uuid.UUID) (usecase.NotificationCreatePersonalParams, error) {
	notifType := domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return usecase.NotificationCreatePersonalParams{
		UserID:  userID,
		Title:   req.Title,
		Content: req.Content,
		Type:    notifType,
	}, nil
}

func UpdateNotificationRequestToParams(req *openapi.UpdateNotificationRequest, ID uuid.UUID) (usecase.NotificationUpdateParams, error) {
	notifType := domain.NotificationInfo

	if req.Type != nil {
		if t, ok := validNotificationType(string(*req.Type)); ok {
			notifType = t
		}
	}

	return usecase.NotificationUpdateParams{
		ID:       ID,
		Title:    req.Title,
		Content:  req.Content,
		Type:     notifType,
		IsPinned: lo.FromPtrOr(req.IsPinned, false),
	}, nil
}
