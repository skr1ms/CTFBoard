package notification

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	notifMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification/mock"
)

// mockBroadcaster is a simple test double for NotificationBroadcaster.
type mockBroadcaster struct {
	calls []struct{ message, level string }
}

func (m *mockBroadcaster) NotifyNotification(message, level string) {
	m.calls = append(m.calls, struct{ message, level string }{message, level})
}

func newUC(t *testing.T) (*NotificationUseCase, *notifMock.MockNotificationRepository, *mockBroadcaster) {
	t.Helper()

	repo := notifMock.NewMockNotificationRepository(t)
	broadcaster := &mockBroadcaster{}
	uc := NewNotificationUseCase(NotificationDeps{
		NotifRepo:   repo,
		Broadcaster: broadcaster,
	})

	return uc, repo, broadcaster
}

func notificationGlobalParams(title, content string, notifType domain.NotificationType, isPinned bool) usecase.NotificationCreateGlobalParams {
	return usecase.NotificationCreateGlobalParams{
		Title:    title,
		Content:  content,
		Type:     notifType,
		IsPinned: isPinned,
	}
}

func notificationPersonalParams(userID uuid.UUID, title, content string, notifType domain.NotificationType) usecase.NotificationCreatePersonalParams {
	return usecase.NotificationCreatePersonalParams{
		UserID:  userID,
		Title:   title,
		Content: content,
		Type:    notifType,
	}
}

func notificationUpdateParams(ID uuid.UUID, title, content string, notifType domain.NotificationType, isPinned bool) usecase.NotificationUpdateParams {
	return usecase.NotificationUpdateParams{
		ID:       ID,
		Title:    title,
		Content:  content,
		Type:     notifType,
		IsPinned: isPinned,
	}
}

func TestCreateGlobal_Success(t *testing.T) {
	t.Parallel()

	uc, repo, broadcaster := newUC(t)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Title == "title" && n.Content == "content" && n.Type == domain.NotificationInfo && n.IsGlobal
	})).Return(nil)

	notif, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "content", domain.NotificationInfo, false))

	require.NoError(t, err)
	require.NotNil(t, notif)
	assert.Equal(t, "title", notif.Title)
	assert.True(t, notif.IsGlobal)
	assert.Len(t, broadcaster.calls, 1)
	assert.Equal(t, "title", broadcaster.calls[0].message)
	assert.Equal(t, "info", broadcaster.calls[0].level)
}

func TestCreateGlobal_EmptyTitle_Error(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUC(t)

	_, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("", "content", domain.NotificationInfo, false))

	require.ErrorIs(t, err, apperr.ErrNotificationTitleContentRequired)
}

func TestCreateGlobal_EmptyContent_Error(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUC(t)

	_, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "", domain.NotificationInfo, false))

	require.ErrorIs(t, err, apperr.ErrNotificationTitleContentRequired)
}

func TestCreateGlobal_InvalidType_Error(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUC(t)

	_, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "content", domain.NotificationType("invalid"), false))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notification type")
}

func TestCreateGlobal_NilBroadcaster_Success(t *testing.T) {
	t.Parallel()

	repo := notifMock.NewMockNotificationRepository(t)
	uc := NewNotificationUseCase(NotificationDeps{NotifRepo: repo})
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	notif, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "content", domain.NotificationInfo, false))

	require.NoError(t, err)
	require.NotNil(t, notif)
}

func TestCreateGlobal_RepoError(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(errors.New("db error"))

	_, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "content", domain.NotificationInfo, false))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "db error")
}

func TestCreatePersonal_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	userID := uuid.New()

	repo.EXPECT().CreateUserNotification(mock.Anything, mock.MatchedBy(func(n *domain.UserNotification) bool {
		return n.UserID == userID && n.Title == "personal" && n.Content == "body" && !n.IsRead
	})).Return(nil)

	notif, err := uc.CreatePersonal(context.Background(), notificationPersonalParams(userID, "personal", "body", domain.NotificationWarning))

	require.NoError(t, err)
	require.NotNil(t, notif)
	assert.Equal(t, userID, notif.UserID)
	assert.False(t, notif.IsRead)
}

func TestCreatePersonal_EmptyContent_Error(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUC(t)

	_, err := uc.CreatePersonal(context.Background(), notificationPersonalParams(uuid.New(), "title", "", domain.NotificationInfo))

	require.ErrorIs(t, err, apperr.ErrNotificationTitleContentRequired)
}

func TestGetGlobal_PaginationOffset(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	want := []*domain.Notification{{ID: uuid.New(), Title: "n1"}}
	// page=3, perPage=10 => offset=20
	repo.EXPECT().GetAll(mock.Anything, 10, 20).Return(want, nil)

	got, err := uc.GetGlobal(context.Background(), 3, 10)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestGetUserNotifications_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	userID := uuid.New()
	want := []*domain.UserNotification{{ID: uuid.New(), UserID: userID}}
	// page=2, perPage=5 => offset=5
	repo.EXPECT().GetUserNotifications(mock.Anything, userID, 5, 5).Return(want, nil)

	got, err := uc.GetUserNotifications(context.Background(), userID, 2, 5)

	require.NoError(t, err)
	assert.Equal(t, want, got)
}

func TestMarkAsRead_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	notifID := uuid.New()
	userID := uuid.New()
	userNotif := &domain.UserNotification{ID: notifID, UserID: userID}

	repo.EXPECT().GetUserNotificationByID(mock.Anything, notifID, userID).Return(userNotif, nil)
	repo.EXPECT().MarkAsRead(mock.Anything, notifID, userID).Return(nil)

	err := uc.MarkAsRead(context.Background(), notifID, userID)

	require.NoError(t, err)
}

func TestMarkAsRead_NotFound_Error(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	notifID := uuid.New()
	userID := uuid.New()

	repo.EXPECT().GetUserNotificationByID(mock.Anything, notifID, userID).Return(nil, apperr.ErrNotificationNotFound)

	err := uc.MarkAsRead(context.Background(), notifID, userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "GetUserNotificationByID")
}

func TestCountUnread_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	userID := uuid.New()
	repo.EXPECT().CountUnread(mock.Anything, userID).Return(7, nil)

	count, err := uc.CountUnread(context.Background(), userID)

	require.NoError(t, err)
	assert.Equal(t, 7, count)
}

func TestUpdate_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	id := uuid.New()
	existing := &domain.Notification{ID: id, Title: "old", Content: "old content", Type: domain.NotificationInfo}

	repo.EXPECT().GetByID(mock.Anything, id).Return(existing, nil)
	repo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.Title == "new title" && n.Content == "new content" && n.Type == domain.NotificationSuccess
	})).Return(nil)

	result, err := uc.Update(context.Background(), notificationUpdateParams(id, "new title", "new content", domain.NotificationSuccess, true))

	require.NoError(t, err)
	assert.Equal(t, "new title", result.Title)
	assert.Equal(t, domain.NotificationSuccess, result.Type)
}

func TestUpdate_InvalidType_Error(t *testing.T) {
	t.Parallel()

	uc, _, _ := newUC(t)

	_, err := uc.Update(context.Background(), notificationUpdateParams(uuid.New(), "t", "c", domain.NotificationType("bad"), false))

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid notification type")
}

func TestDelete_Success(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	id := uuid.New()
	repo.EXPECT().Delete(mock.Anything, id).Return(nil)

	err := uc.Delete(context.Background(), id)

	require.NoError(t, err)
}

func TestCreateGlobal_IsPinned(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	repo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(n *domain.Notification) bool {
		return n.IsPinned
	})).Return(nil)

	notif, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("pinned", "content", domain.NotificationInfo, true))

	require.NoError(t, err)
	assert.True(t, notif.IsPinned)
}

func TestCreateGlobal_HasTimestamp(t *testing.T) {
	t.Parallel()

	uc, repo, _ := newUC(t)
	before := time.Now().Add(-time.Second)

	repo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil)

	notif, err := uc.CreateGlobal(context.Background(), notificationGlobalParams("title", "content", domain.NotificationInfo, false))

	require.NoError(t, err)
	assert.True(t, notif.CreatedAt.After(before))
}
