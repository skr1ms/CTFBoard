package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestSettingsRepo_Get_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.SettingsRepo.Get(ctx)

	require.NoError(t, err)
	assert.Equal(t, 1, settings.ID)
	assert.Equal(t, "AstroCTFb", settings.AppName)
	assert.True(t, settings.VerifyEmails)
	assert.Equal(t, "http://localhost:3000", settings.FrontendURL)
	assert.Equal(t, 24, settings.VerifyTTLHours)
	assert.Equal(t, 1, settings.ResetTTLHours)
	assert.Equal(t, 10, settings.SubmitLimitPerUser)
	assert.Equal(t, 1, settings.SubmitLimitDurationMin)
	assert.Equal(t, domain.ScoreboardVisiblePublic, settings.ScoreboardVisible)
	assert.True(t, settings.RegistrationOpen)
}

func TestSettingsRepo_Update_Success(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.AppName = "Updated AstroCTFb"
	settings.VerifyEmails = false
	settings.FrontendURL = "https://ctf.example.com"
	settings.VerifyTTLHours = 48
	settings.ResetTTLHours = 2
	settings.SubmitLimitPerUser = 20
	settings.SubmitLimitDurationMin = 5
	settings.ScoreboardVisible = domain.ScoreboardVisibleHidden
	settings.RegistrationOpen = false

	err = f.SettingsRepo.Update(ctx, settings)
	require.NoError(t, err)

	updated, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	assert.Equal(t, "Updated AstroCTFb", updated.AppName)
	assert.False(t, updated.VerifyEmails)
	assert.Equal(t, "https://ctf.example.com", updated.FrontendURL)
	assert.Equal(t, 48, updated.VerifyTTLHours)
	assert.Equal(t, 2, updated.ResetTTLHours)
	assert.Equal(t, 20, updated.SubmitLimitPerUser)
	assert.Equal(t, 5, updated.SubmitLimitDurationMin)
	assert.Equal(t, domain.ScoreboardVisibleHidden, updated.ScoreboardVisible)
	assert.False(t, updated.RegistrationOpen)
}

func TestSettingsRepo_Update_ScoreboardVisibility(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.ScoreboardVisible = domain.ScoreboardVisibleAdminsOnly
	err = f.SettingsRepo.Update(ctx, settings)
	require.NoError(t, err)

	updated, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)
	assert.Equal(t, domain.ScoreboardVisibleAdminsOnly, updated.ScoreboardVisible)
}

func TestSettingsRepo_Update_InvalidScoreboardVisibility_Error(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx := context.Background()

	settings, err := f.SettingsRepo.Get(ctx)
	require.NoError(t, err)

	settings.ScoreboardVisible = "invalid_value"
	err = f.SettingsRepo.Update(ctx, settings)

	assert.Error(t, err)
}

func TestSettingsRepo_Get_Error_CancelledContext(t *testing.T) {
	testPool := SetupTestPool(t)
	f := NewTestFixture(testPool.Pool)
	f.ResetAppSettings(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := f.SettingsRepo.Get(ctx)

	require.Error(t, err)
}
