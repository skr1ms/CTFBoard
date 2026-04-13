package integration_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestNotificationRepo_Create_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := &domain.Notification{
		Title:    "Test",
		Content:  "Body",
		Type:     domain.NotificationInfo,
		IsPinned: false,
		IsGlobal: true,
	}
	err := f.NotificationRepo.Create(ctx, notif)
	require.NoError(t, err)
	assert.NotEmpty(t, notif.ID)
}

func TestNotificationRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notif := &domain.Notification{Title: "x", Content: "x", Type: domain.NotificationInfo}
	err := f.NotificationRepo.Create(ctx, notif)
	assert.Error(t, err)
}

func TestNotificationRepo_GetByID_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := f.CreateNotification(t, "gbi")
	got, err := f.NotificationRepo.GetByID(ctx, notif.ID)
	require.NoError(t, err)
	assert.Equal(t, notif.ID, got.ID)
	assert.Equal(t, notif.Title, got.Title)
}

func TestNotificationRepo_GetByID_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.NotificationRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotificationNotFound)
}

func TestNotificationRepo_GetAll_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := f.CreateNotification(t, "ga1")
	list, err := f.NotificationRepo.GetAll(ctx, 10, 0)
	require.NoError(t, err)

	ids := make(map[uuid.UUID]bool)

	for _, n := range list {
		ids[n.ID] = true
	}

	assert.True(t, ids[notif.ID], "notification should be in GetAll result")
}

func TestNotificationRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.NotificationRepo.GetAll(ctx, 10, 0)
	assert.Error(t, err)
}

func TestNotificationRepo_Update_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := f.CreateNotification(t, "upd")
	notif.Title = "Updated"
	err := f.NotificationRepo.Update(ctx, notif)
	require.NoError(t, err)
	got, err := f.NotificationRepo.GetByID(ctx, notif.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
}

func TestNotificationRepo_Update_Error_CancelledContext(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	notif := f.CreateNotification(t, "upderr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.NotificationRepo.Update(ctx, notif)
	assert.Error(t, err)
}

func TestNotificationRepo_Delete_Success(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := f.CreateNotification(t, "del")
	err := f.NotificationRepo.Delete(ctx, notif.ID)
	require.NoError(t, err)
	_, err = f.NotificationRepo.GetByID(ctx, notif.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNotificationNotFound)
}

func TestNotificationRepo_Delete_Error_NotFound(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.NotificationRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}

func TestNotificationRepo_CreateUserNotification(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "unnotif")

	userNotif := &domain.UserNotification{
		UserID:  user.ID,
		Title:   "personal title",
		Content: "personal content",
		Type:    domain.NotificationInfo,
		IsRead:  false,
	}
	err := f.NotificationRepo.CreateUserNotification(ctx, userNotif)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, userNotif.ID)
}

func TestNotificationRepo_GetUserNotifications(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "ung")

	for range 3 {
		un := &domain.UserNotification{
			UserID:  user.ID,
			Title:   "t",
			Content: "c",
			Type:    domain.NotificationInfo,
		}
		require.NoError(t, f.NotificationRepo.CreateUserNotification(ctx, un))
	}

	list, err := f.NotificationRepo.GetUserNotifications(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 3)

	for _, n := range list {
		assert.Equal(t, user.ID, n.UserID)
	}
}

func TestNotificationRepo_MarkAsRead(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "unread")

	un := &domain.UserNotification{
		UserID:  user.ID,
		Title:   "unread",
		Content: "content",
		Type:    domain.NotificationInfo,
	}
	require.NoError(t, f.NotificationRepo.CreateUserNotification(ctx, un))

	got, err := f.NotificationRepo.GetUserNotificationByID(ctx, un.ID, user.ID)
	require.NoError(t, err)
	assert.False(t, got.IsRead)

	require.NoError(t, f.NotificationRepo.MarkAsRead(ctx, un.ID, user.ID))

	got2, err := f.NotificationRepo.GetUserNotificationByID(ctx, un.ID, user.ID)
	require.NoError(t, err)
	assert.True(t, got2.IsRead)
}

func TestNotificationRepo_CountUnread(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "uncnt")

	for range 3 {
		un := &domain.UserNotification{
			UserID:  user.ID,
			Title:   "t",
			Content: "c",
			Type:    domain.NotificationInfo,
		}
		require.NoError(t, f.NotificationRepo.CreateUserNotification(ctx, un))
	}

	count, err := f.NotificationRepo.CountUnread(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 3, count)

	// read one
	list, err := f.NotificationRepo.GetUserNotifications(ctx, user.ID, 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, list)
	require.NoError(t, f.NotificationRepo.MarkAsRead(ctx, list[0].ID, user.ID))

	count2, err := f.NotificationRepo.CountUnread(ctx, user.ID)
	require.NoError(t, err)
	assert.Equal(t, 2, count2)
}

func TestNotificationRepo_DeleteUserNotification(t *testing.T) {
	t.Parallel()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	user := f.CreateUser(t, "undel")

	un := &domain.UserNotification{
		UserID:  user.ID,
		Title:   "delete me",
		Content: "content",
		Type:    domain.NotificationInfo,
	}
	require.NoError(t, f.NotificationRepo.CreateUserNotification(ctx, un))

	require.NoError(t, f.NotificationRepo.DeleteUserNotification(ctx, un.ID, user.ID))

	_, err := f.NotificationRepo.GetUserNotificationByID(ctx, un.ID, user.ID)
	assert.Error(t, err)
}
