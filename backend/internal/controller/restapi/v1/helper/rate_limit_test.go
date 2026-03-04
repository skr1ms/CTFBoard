package helper

import (
	"context"
	"errors"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/v1/helper/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetRateLimitConfig_Success(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsGetter(t)
	repo.On("Get", context.Background()).Return(&entity.Settings{
		RateLimitLoginPerMinute:          20,
		RateLimitRegisterPerMinute:       8,
		RateLimitForgotPasswordPerMinute: 5,
		RateLimitResetPasswordPerMinute:  7,
		RateLimitLogoutPerMinute:         15,
		RateLimitRefreshPerMinute:        12,
		RateLimitScoreboardPerMinute:     60,
		RateLimitGeneralIPPerMinute:      200,
		RateLimitVerifyEmailPerMinute:    15,
		RateLimitOAuthCallbackPerMinute:  25,
	}, nil)

	cfg, err := GetRateLimitConfig(context.Background(), repo)
	require.NoError(t, err)
	assert.Equal(t, 20, cfg.LoginPerMinute)
	assert.Equal(t, 8, cfg.RegisterPerMinute)
	assert.Equal(t, 5, cfg.ForgotPasswordPerMinute)
	assert.Equal(t, 7, cfg.ResetPasswordPerMinute)
	assert.Equal(t, 15, cfg.LogoutPerMinute)
	assert.Equal(t, 12, cfg.RefreshPerMinute)
	assert.Equal(t, 60, cfg.ScoreboardPerMinute)
	assert.Equal(t, 200, cfg.GeneralIPPerMinute)
	assert.Equal(t, 15, cfg.VerifyEmailPerMinute)
	assert.Equal(t, 25, cfg.OAuthCallbackPerMinute)
}

func TestGetRateLimitConfig_ZeroValues_UsesDefaults(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsGetter(t)
	repo.On("Get", context.Background()).Return(&entity.Settings{}, nil)

	cfg, err := GetRateLimitConfig(context.Background(), repo)
	require.NoError(t, err)
	assert.Equal(t, 10, cfg.LoginPerMinute)
	assert.Equal(t, 5, cfg.RegisterPerMinute)
	assert.Equal(t, 3, cfg.ForgotPasswordPerMinute)
	assert.Equal(t, 5, cfg.ResetPasswordPerMinute)
	assert.Equal(t, 10, cfg.LogoutPerMinute)
	assert.Equal(t, 10, cfg.RefreshPerMinute)
	assert.Equal(t, 30, cfg.ScoreboardPerMinute)
	assert.Equal(t, 100, cfg.GeneralIPPerMinute)
	assert.Equal(t, 10, cfg.VerifyEmailPerMinute)
	assert.Equal(t, 20, cfg.OAuthCallbackPerMinute)
}

func TestGetRateLimitConfig_RepoError_ReturnsError(t *testing.T) {
	t.Parallel()
	repo := mocks.NewMockSettingsGetter(t)
	repo.On("Get", context.Background()).Return((*entity.Settings)(nil), errors.New("db error"))

	cfg, err := GetRateLimitConfig(context.Background(), repo)
	assert.Error(t, err)
	assert.Nil(t, cfg)
}
