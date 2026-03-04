package notification

import (
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification/mocks"
	"github.com/google/uuid"
)

type NotificationTestHelper struct {
	t    *testing.T
	deps *notificationTestDeps
}

type notificationTestDeps struct {
	notifRepo *mocks.MockNotificationRepository
}

func NewNotificationTestHelper(t *testing.T) *NotificationTestHelper {
	t.Helper()
	return &NotificationTestHelper{
		t: t,
		deps: &notificationTestDeps{
			notifRepo: mocks.NewMockNotificationRepository(t),
		},
	}
}

func (h *NotificationTestHelper) Deps() *notificationTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *NotificationTestHelper) CreateUseCase() *NotificationUseCase {
	h.t.Helper()
	return NewNotificationUseCase(NotificationDeps{NotifRepo: h.deps.notifRepo})
}

func (h *NotificationTestHelper) NewNotification(title, content string, notifType entity.NotificationType, isPinned, isGlobal bool) *entity.Notification {
	h.t.Helper()
	return &entity.Notification{
		ID:        uuid.New(),
		Title:     title,
		Content:   content,
		Type:      notifType,
		IsPinned:  isPinned,
		IsGlobal:  isGlobal,
		CreatedAt: time.Now(),
	}
}

func (h *NotificationTestHelper) NewUserNotification(userID uuid.UUID, title, content string, notifType entity.NotificationType) *entity.UserNotification {
	h.t.Helper()
	return &entity.UserNotification{
		ID:        uuid.New(),
		UserID:    userID,
		Title:     title,
		Content:   content,
		Type:      notifType,
		IsRead:    false,
		CreatedAt: time.Now(),
	}
}
