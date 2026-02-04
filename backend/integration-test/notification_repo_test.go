package integration_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNotificationRepo_Create_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := &entity.Notification{
		Title:    "Test",
		Content:  "Body",
		Type:     entity.NotificationInfo,
		IsPinned: false,
		IsGlobal: true,
	}
	err := f.NotificationRepo.Create(ctx, notif)
	require.NoError(t, err)
	assert.NotEmpty(t, notif.ID)
}

func TestNotificationRepo_Create_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	notif := &entity.Notification{Title: "x", Content: "x", Type: entity.NotificationInfo}
	err := f.NotificationRepo.Create(ctx, notif)
	assert.Error(t, err)
}

func TestNotificationRepo_GetByID_Success(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	_, err := f.NotificationRepo.GetByID(ctx, uuid.New())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotificationNotFound))
}

func TestNotificationRepo_GetAll_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	f.CreateNotification(t, "ga1")
	list, err := f.NotificationRepo.GetAll(ctx, 10, 0)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(list), 1)
}

func TestNotificationRepo_GetAll_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.NotificationRepo.GetAll(ctx, 10, 0)
	assert.Error(t, err)
}

func TestNotificationRepo_Update_Success(t *testing.T) {
	t.Helper()
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
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	notif := f.CreateNotification(t, "upderr")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := f.NotificationRepo.Update(ctx, notif)
	assert.Error(t, err)
}

func TestNotificationRepo_Delete_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	notif := f.CreateNotification(t, "del")
	err := f.NotificationRepo.Delete(ctx, notif.ID)
	require.NoError(t, err)
	_, err = f.NotificationRepo.GetByID(ctx, notif.ID)
	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNotificationNotFound))
}

func TestNotificationRepo_Delete_Error_NotFound(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	ctx := context.Background()

	err := f.NotificationRepo.Delete(ctx, uuid.New())
	assert.NoError(t, err)
}
