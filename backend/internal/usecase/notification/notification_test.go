package notification

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/notification/mocks"
)

type notificationTestDeps struct {
	notifRepo *mocks.MockNotificationRepository
}

func newNotificationTestDeps(t *testing.T) *notificationTestDeps {
	t.Helper()
	return &notificationTestDeps{notifRepo: mocks.NewMockNotificationRepository(t)}
}

func (d *notificationTestDeps) createUseCase() *NotificationUseCase {
	return NewNotificationUseCase(NotificationDeps{NotifRepo: d.notifRepo})
}

func newTestNotification(title, content string, notifType entity.NotificationType, isPinned, isGlobal bool) *entity.Notification {
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

func newTestUserNotification(userID uuid.UUID, title, content string, notifType entity.NotificationType) *entity.UserNotification {
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

func TestNotificationUseCase_CreateGlobal_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	title, content := "Title", "Content"
	notifType := entity.NotificationInfo

	d.notifRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.Notification) {
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
		assert.True(t, n.IsPinned)
		assert.True(t, n.IsGlobal)
	})

	uc := d.createUseCase()
	got, err := uc.CreateGlobal(ctx, title, content, notifType, true)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, title, got.Title)
}

func TestNotificationUseCase_CreateGlobal_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()

	d.notifRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	got, err := uc.CreateGlobal(ctx, "T", "C", entity.NotificationInfo, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_CreatePersonal_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	title, content := "Title", "Content"
	notifType := entity.NotificationWarning

	d.notifRepo.EXPECT().CreateUserNotification(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.UserNotification) {
		assert.Equal(t, userID, n.UserID)
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
	})

	uc := d.createUseCase()
	got, err := uc.CreatePersonal(ctx, userID, title, content, notifType)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, userID, got.UserID)
}

func TestNotificationUseCase_CreatePersonal_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()

	d.notifRepo.EXPECT().CreateUserNotification(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := d.createUseCase()
	got, err := uc.CreatePersonal(ctx, userID, "T", "C", entity.NotificationInfo)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_GetGlobal_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	list := []*entity.Notification{newTestNotification("T", "C", entity.NotificationInfo, false, true)}

	d.notifRepo.EXPECT().GetAll(mock.Anything, 20, 0).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetGlobal(ctx, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestNotificationUseCase_GetGlobal_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()

	d.notifRepo.EXPECT().GetAll(mock.Anything, 20, 0).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetGlobal(ctx, 1, 20)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_GetUserNotifications_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	list := []*entity.UserNotification{newTestUserNotification(userID, "T", "C", entity.NotificationInfo)}

	d.notifRepo.EXPECT().GetUserNotifications(mock.Anything, userID, 20, 0).Return(list, nil)

	uc := d.createUseCase()
	got, err := uc.GetUserNotifications(ctx, userID, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestNotificationUseCase_GetUserNotifications_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()

	d.notifRepo.EXPECT().GetUserNotifications(mock.Anything, userID, 20, 0).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.GetUserNotifications(ctx, userID, 1, 20)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_MarkAsRead_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	d.notifRepo.EXPECT().GetUserNotificationByID(mock.Anything, id, userID).Return(&entity.UserNotification{ID: id, UserID: userID}, nil)
	d.notifRepo.EXPECT().MarkAsRead(mock.Anything, id, userID).Return(nil)

	uc := d.createUseCase()
	err := uc.MarkAsRead(ctx, id, userID)

	assert.NoError(t, err)
}

func TestNotificationUseCase_MarkAsRead_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	d.notifRepo.EXPECT().GetUserNotificationByID(mock.Anything, id, userID).Return(&entity.UserNotification{ID: id, UserID: userID}, nil)
	d.notifRepo.EXPECT().MarkAsRead(mock.Anything, id, userID).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.MarkAsRead(ctx, id, userID)

	assert.Error(t, err)
}

func TestNotificationUseCase_CountUnread_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()
	count := 5

	d.notifRepo.EXPECT().CountUnread(mock.Anything, userID).Return(count, nil)

	uc := d.createUseCase()
	got, err := uc.CountUnread(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, count, got)
}

func TestNotificationUseCase_CountUnread_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	userID := uuid.New()

	d.notifRepo.EXPECT().CountUnread(mock.Anything, userID).Return(0, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.CountUnread(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, 0, got)
}

func TestNotificationUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id := uuid.New()
	notif := newTestNotification("Old", "OldC", entity.NotificationInfo, false, true)
	notif.ID = id
	title, content := "New", "NewC"
	notifType := entity.NotificationWarning

	d.notifRepo.EXPECT().GetByID(mock.Anything, id).Return(notif, nil)
	d.notifRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.Notification) {
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
		assert.True(t, n.IsPinned)
	})

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, title, content, notifType, true)

	assert.NoError(t, err)
	assert.Equal(t, title, got.Title)
}

func TestNotificationUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.notifRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := d.createUseCase()
	got, err := uc.Update(ctx, id, "T", "C", entity.NotificationInfo, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_Delete_Success(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.notifRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestNotificationUseCase_Delete_Error(t *testing.T) {
	t.Parallel()
	d := newNotificationTestDeps(t)
	ctx := context.Background()
	id := uuid.New()

	d.notifRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := d.createUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
