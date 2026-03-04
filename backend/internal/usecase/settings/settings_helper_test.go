package settings

import (
	"context"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/go-redis/redismock/v9"
	"github.com/stretchr/testify/mock"
)

type SettingsTestHelper struct {
	t    *testing.T
	deps *settingsTestDeps
}

type settingsTestDeps struct {
	SettingsRepo *mocks.MockSettingsRepository
	auditLogRepo *mocks.MockAuditLogRepository
	tm           *mocks.MockTransactionManager
}

func NewSettingsTestHelper(t *testing.T) *SettingsTestHelper {
	t.Helper()
	tm := mocks.NewMockTransactionManager(t)
	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Maybe()
	return &SettingsTestHelper{
		t: t,
		deps: &settingsTestDeps{
			SettingsRepo: mocks.NewMockSettingsRepository(t),
			auditLogRepo: mocks.NewMockAuditLogRepository(t),
			tm:           tm,
		},
	}
}

func (h *SettingsTestHelper) Deps() *settingsTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *SettingsTestHelper) CreateSettingsUseCase() (*SettingsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSettingsUseCase(SettingsDeps{
		Repo:         h.deps.SettingsRepo,
		AuditLogRepo: h.deps.auditLogRepo,
		TM:           h.deps.tm,
		Redis:        &cache.RedisKeyValueStore{Client: client},
	}), redis
}

func (h *SettingsTestHelper) NewAppSettings() *entity.Settings {
	h.t.Helper()
	return &entity.Settings{
		ID:                               1,
		AppName:                          "AstroCTFb",
		VerifyEmails:                     true,
		FrontendURL:                      "http://localhost:3000",
		CORSOrigins:                      "http://localhost:3000",
		ResendEnabled:                    false,
		ResendFromEmail:                  "noreply@astroctfb.local",
		ResendFromName:                   "AstroCTFb",
		VerifyTTLHours:                   24,
		ResetTTLHours:                    1,
		SubmitLimitPerUser:               10,
		SubmitLimitDurationMin:           1,
		ScoreboardVisible:                entity.ScoreboardVisiblePublic,
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
		WriteupEnabled:                   true,
		UpdatedAt:                        time.Now(),
	}
}

func (h *SettingsTestHelper) NewAppSettingsWithValues(
	submitLimit int,
	submitDuration int,
	verifyTTL int,
	resetTTL int,
	visibility string,
) *entity.Settings {
	h.t.Helper()
	s := h.NewAppSettings()
	s.SubmitLimitPerUser = submitLimit
	s.SubmitLimitDurationMin = submitDuration
	s.VerifyTTLHours = verifyTTL
	s.ResetTTLHours = resetTTL
	s.ScoreboardVisible = visibility
	return s
}
