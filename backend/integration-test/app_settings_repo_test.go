package integration_test

import (
	"context"
	"testing"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppSettingsRepo_Get_Success(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.AppSettingsRepo.Get(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, settings.ID)
	assert.Equal(t, "CTFBoard", settings.AppName)
	assert.True(t, settings.VerifyEmails)
	assert.Equal(t, "http://localhost:3000", settings.FrontendURL)
	assert.Equal(t, 24, settings.VerifyTTLHours)
	assert.Equal(t, 1, settings.ResetTTLHours)
	assert.Equal(t, 10, settings.SubmitLimitPerUser)
	assert.Equal(t, 1, settings.SubmitLimitDurationMin)
	assert.Equal(t, entity.ScoreboardVisiblePublic, settings.ScoreboardVisible)
	assert.True(t, settings.RegistrationOpen)
}

func TestAppSettingsRepo_Update_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.AppName = "Updated CTFBoard"
	settings.VerifyEmails = false
	settings.FrontendURL = "https://ctf.example.com"
	settings.VerifyTTLHours = 48
	settings.ResetTTLHours = 2
	settings.SubmitLimitPerUser = 20
	settings.SubmitLimitDurationMin = 5
	settings.ScoreboardVisible = entity.ScoreboardVisibleHidden
	settings.RegistrationOpen = false

	err = f.AppSettingsRepo.Update(ctx, settings)
	require.NoError(t, err)

	updated, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "Updated CTFBoard", updated.AppName)
	assert.False(t, updated.VerifyEmails)
	assert.Equal(t, "https://ctf.example.com", updated.FrontendURL)
	assert.Equal(t, 48, updated.VerifyTTLHours)
	assert.Equal(t, 2, updated.ResetTTLHours)
	assert.Equal(t, 20, updated.SubmitLimitPerUser)
	assert.Equal(t, 5, updated.SubmitLimitDurationMin)
	assert.Equal(t, entity.ScoreboardVisibleHidden, updated.ScoreboardVisible)
	assert.False(t, updated.RegistrationOpen)
}

func TestAppSettingsRepo_Update_ScoreboardVisibility(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.ScoreboardVisible = entity.ScoreboardVisibleAdminsOnly
	err = f.AppSettingsRepo.Update(ctx, settings)
	require.NoError(t, err)

	updated, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, entity.ScoreboardVisibleAdminsOnly, updated.ScoreboardVisible)
}

func TestAppSettingsRepo_Update_InvalidScoreboardVisibility_Error(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.AppSettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.ScoreboardVisible = "invalid_value"
	err = f.AppSettingsRepo.Update(ctx, settings)

	assert.Error(t, err)
}

func TestAppSettingsRepo_Get_Error_CancelledContext(t *testing.T) {
	t.Helper()
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.AppSettingsRepo.Get(ctx)

	require.Error(t, err)
}
