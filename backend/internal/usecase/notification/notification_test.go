package notification

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestNotificationUseCase_CreateGlobal_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	title, content := "Title", "Content"
	notifType := entity.NotificationInfo
	isPinned := true

	deps.notifRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.Notification) {
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
		assert.True(t, n.IsPinned)
		assert.True(t, n.IsGlobal)
	})

	uc := h.CreateUseCase()
	got, err := uc.CreateGlobal(ctx, title, content, notifType, isPinned)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, title, got.Title)
}

func TestNotificationUseCase_CreateGlobal_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.notifRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.CreateGlobal(ctx, "T", "C", entity.NotificationInfo, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_CreatePersonal_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	title, content := "Title", "Content"
	notifType := entity.NotificationWarning

	deps.notifRepo.EXPECT().CreateUserNotification(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.UserNotification) {
		assert.Equal(t, userID, n.UserID)
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
	})

	uc := h.CreateUseCase()
	got, err := uc.CreatePersonal(ctx, userID, title, content, notifType)

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, userID, got.UserID)
}

func TestNotificationUseCase_CreatePersonal_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()

	deps.notifRepo.EXPECT().CreateUserNotification(mock.Anything, mock.Anything).Return(assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.CreatePersonal(ctx, userID, "T", "C", entity.NotificationInfo)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_GetGlobal_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	list := []*entity.Notification{h.NewNotification("T", "C", entity.NotificationInfo, false, true)}

	deps.notifRepo.EXPECT().GetAll(mock.Anything, 20, 0).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetGlobal(ctx, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestNotificationUseCase_GetGlobal_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()

	deps.notifRepo.EXPECT().GetAll(mock.Anything, 20, 0).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetGlobal(ctx, 1, 20)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_GetUserNotifications_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	list := []*entity.UserNotification{h.NewUserNotification(userID, "T", "C", entity.NotificationInfo)}

	deps.notifRepo.EXPECT().GetUserNotifications(mock.Anything, userID, 20, 0).Return(list, nil)

	uc := h.CreateUseCase()
	got, err := uc.GetUserNotifications(ctx, userID, 1, 20)

	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestNotificationUseCase_GetUserNotifications_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()

	deps.notifRepo.EXPECT().GetUserNotifications(mock.Anything, userID, 20, 0).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.GetUserNotifications(ctx, userID, 1, 20)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_MarkAsRead_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	deps.notifRepo.EXPECT().MarkAsRead(mock.Anything, id, userID).Return(nil)

	uc := h.CreateUseCase()
	err := uc.MarkAsRead(ctx, id, userID)

	assert.NoError(t, err)
}

func TestNotificationUseCase_MarkAsRead_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id, userID := uuid.New(), uuid.New()

	deps.notifRepo.EXPECT().MarkAsRead(mock.Anything, id, userID).Return(assert.AnError)

	uc := h.CreateUseCase()
	err := uc.MarkAsRead(ctx, id, userID)

	assert.Error(t, err)
}

func TestNotificationUseCase_CountUnread_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()
	count := 5

	deps.notifRepo.EXPECT().CountUnread(mock.Anything, userID).Return(count, nil)

	uc := h.CreateUseCase()
	got, err := uc.CountUnread(ctx, userID)

	assert.NoError(t, err)
	assert.Equal(t, count, got)
}

func TestNotificationUseCase_CountUnread_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	userID := uuid.New()

	deps.notifRepo.EXPECT().CountUnread(mock.Anything, userID).Return(0, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.CountUnread(ctx, userID)

	assert.Error(t, err)
	assert.Equal(t, 0, got)
}

func TestNotificationUseCase_Update_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()
	notif := h.NewNotification("Old", "OldC", entity.NotificationInfo, false, true)
	notif.ID = id
	title, content := "New", "NewC"
	notifType := entity.NotificationWarning
	isPinned := true

	deps.notifRepo.EXPECT().GetByID(mock.Anything, id).Return(notif, nil)
	deps.notifRepo.EXPECT().Update(mock.Anything, mock.Anything).Return(nil).Run(func(_ context.Context, n *entity.Notification) {
		assert.Equal(t, title, n.Title)
		assert.Equal(t, content, n.Content)
		assert.Equal(t, notifType, n.Type)
		assert.True(t, n.IsPinned)
	})

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, title, content, notifType, isPinned)

	assert.NoError(t, err)
	assert.Equal(t, title, got.Title)
}

func TestNotificationUseCase_Update_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.notifRepo.EXPECT().GetByID(mock.Anything, id).Return(nil, assert.AnError)

	uc := h.CreateUseCase()
	got, err := uc.Update(ctx, id, "T", "C", entity.NotificationInfo, false)

	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestNotificationUseCase_Delete_Success(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.notifRepo.EXPECT().Delete(mock.Anything, id).Return(nil)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.NoError(t, err)
}

func TestNotificationUseCase_Delete_Error(t *testing.T) {
	h := NewNotificationTestHelper(t)
	deps := h.Deps()
	ctx := context.Background()
	id := uuid.New()

	deps.notifRepo.EXPECT().Delete(mock.Anything, id).Return(assert.AnError)

	uc := h.CreateUseCase()
	err := uc.Delete(ctx, id)

	assert.Error(t, err)
}
