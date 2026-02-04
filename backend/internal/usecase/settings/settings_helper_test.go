package settings

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition/mocks"
)

type SettingsTestHelper struct {
	t    *testing.T
	deps *settingsTestDeps
}

type settingsTestDeps struct {
	appSettingsRepo *mocks.MockAppSettingsRepository
	auditLogRepo    *mocks.MockAuditLogRepository
}

func NewSettingsTestHelper(t *testing.T) *SettingsTestHelper {
	t.Helper()
	return &SettingsTestHelper{
		t: t,
		deps: &settingsTestDeps{
			appSettingsRepo: mocks.NewMockAppSettingsRepository(t),
			auditLogRepo:    mocks.NewMockAuditLogRepository(t),
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
	return NewSettingsUseCase(h.deps.appSettingsRepo, h.deps.auditLogRepo, client), redis
}

func (h *SettingsTestHelper) NewAppSettings() *entity.AppSettings {
	h.t.Helper()
	return &entity.AppSettings{
		ID:                     1,
		AppName:                "CTFBoard",
		VerifyEmails:           true,
		FrontendURL:            "http://localhost:3000",
		CORSOrigins:            "http://localhost:3000",
		ResendEnabled:          false,
		ResendFromEmail:        "noreply@ctfboard.local",
		ResendFromName:         "CTFBoard",
		VerifyTTLHours:         24,
		ResetTTLHours:          1,
		SubmitLimitPerUser:     10,
		SubmitLimitDurationMin: 1,
		ScoreboardVisible:      entity.ScoreboardVisiblePublic,
		RegistrationOpen:       true,
		UpdatedAt:              time.Now(),
	}
}

func (h *SettingsTestHelper) NewAppSettingsWithValues(
	submitLimit int,
	submitDuration int,
	verifyTTL int,
	resetTTL int,
	visibility string,
) *entity.AppSettings {
	h.t.Helper()
	s := h.NewAppSettings()
	s.SubmitLimitPerUser = submitLimit
	s.SubmitLimitDurationMin = submitDuration
	s.VerifyTTLHours = verifyTTL
	s.ResetTTLHours = resetTTL
	s.ScoreboardVisible = visibility
	return s
}
