package settings

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/mock"
	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	settingsMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/settings/mock"
)

type settingsTestDeps struct {
	SettingsRepo *settingsMock.MockSettingsRepository
	auditLogRepo *settingsMock.MockAuditLogRepository
	tm           *settingsMock.MockTransactionManager
}

func newSettingsTestDeps(t *testing.T) *settingsTestDeps {
	t.Helper()
	tm := settingsMock.NewMockTransactionManager(t)
	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Maybe()

	return &settingsTestDeps{
		SettingsRepo: settingsMock.NewMockSettingsRepository(t),
		auditLogRepo: settingsMock.NewMockAuditLogRepository(t),
		tm:           tm,
	}
}

func (d *settingsTestDeps) createSettingsUseCase(t *testing.T) (*SettingsUseCase, redismock.ClientMock) {
	t.Helper()

	client, redis := redismock.NewClientMock()

	return NewSettingsUseCase(SettingsDeps{
		Repo:         d.SettingsRepo,
		AuditLogRepo: d.auditLogRepo,
		TM:           d.tm,
		Redis:        &cachekit.RedisKeyValueStore{Client: client},
	}), redis
}

func newTestAppSettings() *domain.Settings {
	return &domain.Settings{
		ID:                               1,
		AppName:                          "CTF Platform",
		VerifyEmails:                     true,
		FrontendURL:                      "http://localhost:3000",
		CORSOrigins:                      "http://localhost:3000",
		ResendEnabled:                    false,
		ResendFromEmail:                  "noreply@ctf-platform.local",
		ResendFromName:                   "CTF Platform",
		VerifyTTLHours:                   24,
		ResetTTLHours:                    1,
		SubmitLimitPerUser:               10,
		SubmitLimitDurationMin:           1,
		ScoreboardVisible:                domain.ScoreboardVisiblePublic,
		RegistrationOpen:                 true,
		DefaultPerPage:                   20,
		MaxPerPage:                       100,
		CSVExportMaxRows:                 10000,
		RateLimitLoginPerMinute:          10,
		RateLimitRegisterPerMinute:       5,
		RateLimitForgotPasswordPerMinute: 5,
		RateLimitResetPasswordPerMinute:  5,
		RateLimitLogoutPerMinute:         10,
		RateLimitRefreshPerMinute:        10,
		RateLimitScoreboardPerMinute:     30,
		RateLimitGeneralIPPerMinute:      60,
		RateLimitVerifyEmailPerMinute:    10,
		RateLimitOAuthCallbackPerMinute:  20,
		RateLimitOAuthRedirectPerMinute:  20,
		RateLimitCommentPerMinute:        10,
		WriteupEnabled:                   true,
		UpdatedAt:                        time.Now(),
	}
}

func newTestAppSettingsWithValues(submitLimit, submitDuration, verifyTTL, resetTTL int, visibility string) *domain.Settings {
	s := newTestAppSettings()
	s.SubmitLimitPerUser = submitLimit
	s.SubmitLimitDurationMin = submitDuration
	s.VerifyTTLHours = verifyTTL
	s.ResetTTLHours = resetTTL
	s.ScoreboardVisible = visibility

	return s
}
