package settings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
)

type settingsTestDeps struct {
	SettingsRepo *compMock.MockSettingsRepository
	auditLogRepo *compMock.MockAuditLogRepository
	tm           *compMock.MockTransactionManager
}

func newSettingsTestDeps(t *testing.T) *settingsTestDeps {
	t.Helper()
	tm := compMock.NewMockTransactionManager(t)
	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Maybe()
	return &settingsTestDeps{
		SettingsRepo: compMock.NewMockSettingsRepository(t),
		auditLogRepo: compMock.NewMockAuditLogRepository(t),
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

func TestSettingsUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()

	redisClient.ExpectGet(cache.KeyAppSettings).SetErr(redis.Nil)
	d.SettingsRepo.On("Get", mock.Anything).Return(settings, nil)
	redisClient.Regexp().ExpectSet(cache.KeyAppSettings, `.*`, cacheTTL).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	assert.Equal(t, settings.SubmitLimitPerUser, result.SubmitLimitPerUser)
	assert.Equal(t, settings.ScoreboardVisible, result.ScoreboardVisible)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Get_Cached_Success(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()
	bytes, err := json.Marshal(settings)
	require.NoError(t, err)

	redisClient.ExpectGet(cache.KeyAppSettings).SetVal(string(bytes))

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	d.SettingsRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Get_Error(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	redisClient.ExpectGet(cache.KeyAppSettings).SetErr(redis.Nil)
	d.SettingsRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "SettingsUseCase - Get")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	d.SettingsRepo.On("GetForUpdate", mock.Anything).Return(settings, nil)
	d.SettingsRepo.On("UpdateIfCurrent", mock.Anything, mock.MatchedBy(func(s *domain.Settings) bool {
		return s.ID == settings.ID && s.AppName == settings.AppName
	})).Return(nil)
	redisClient.ExpectDel(cache.KeyAppSettings).SetVal(1)
	d.auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *domain.AuditLog) bool {
		return a.Action == domain.AuditActionUpdate &&
			a.EntityType == domain.AuditEntityAppSettings &&
			a.EntityID == "settings" &&
			a.IP == clientIP &&
			*a.UserID == actorID
	})).Return(nil)

	err := uc.Update(context.Background(), settings, actorID, clientIP)

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()

	d.SettingsRepo.On("GetForUpdate", mock.Anything).Return(settings, nil)
	d.SettingsRepo.On("UpdateIfCurrent", mock.Anything, settings).Return(errors.New("db error"))

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SettingsUseCase - Update")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_AuditLogError(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()

	d.SettingsRepo.On("GetForUpdate", mock.Anything).Return(settings, nil)
	d.SettingsRepo.On("UpdateIfCurrent", mock.Anything, settings).Return(nil)
	d.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("audit error"))

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "AuditLogRepo.Create")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_SubmitLimitPerUser(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(0, 1, 24, 1, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit_limit_per_user must be >= 1")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_SubmitLimitDuration(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 0, 24, 1, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit_limit_duration_min must be >= 1")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_VerifyTTL_TooLow(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 1, 0, 1, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_VerifyTTL_TooHigh(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 1, 200, 1, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ResetTTL_TooLow(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 1, 24, 0, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ResetTTL_TooHigh(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 1, 24, 200, domain.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ScoreboardVisible_Invalid(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettingsWithValues(10, 1, 24, 1, "invalid")

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoreboard_visible must be public, hidden, or admins_only")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ScoreboardVisible_AllValid(t *testing.T) {
	t.Parallel()
	validValues := []string{
		domain.ScoreboardVisiblePublic,
		domain.ScoreboardVisibleHidden,
		domain.ScoreboardVisibleAdminsOnly,
	}

	for _, visibility := range validValues {
		t.Run(visibility, func(t *testing.T) {
			t.Parallel()
			d := newSettingsTestDeps(t)
			uc, redisClient := d.createSettingsUseCase(t)

			settings := newTestAppSettingsWithValues(10, 1, 24, 1, visibility)

			d.SettingsRepo.On("GetForUpdate", mock.Anything).Return(settings, nil)
			d.SettingsRepo.On("UpdateIfCurrent", mock.Anything, settings).Return(nil)
			redisClient.ExpectDel(cache.KeyAppSettings).SetVal(1)
			d.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

			err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

			assert.NoError(t, err)
			assert.NoError(t, redisClient.ExpectationsWereMet())
		})
	}
}

func TestSettingsUseCase_Get_InvalidCachedJSON(t *testing.T) {
	t.Parallel()
	d := newSettingsTestDeps(t)
	uc, redisClient := d.createSettingsUseCase(t)

	settings := newTestAppSettings()

	redisClient.ExpectGet(cache.KeyAppSettings).SetVal("invalid json")
	d.SettingsRepo.On("Get", mock.Anything).Return(settings, nil)
	redisClient.Regexp().ExpectSet(cache.KeyAppSettings, `.*`, cacheTTL).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_BoundaryValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		submitLimit int
		submitDur   int
		verifyTTL   int
		resetTTL    int
		visibility  string
		wantErr     bool
	}{
		{"min valid values", 1, 1, 1, 1, domain.ScoreboardVisiblePublic, false},
		{"max valid TTL", 10, 1, 168, 168, domain.ScoreboardVisiblePublic, false},
		{"verify TTL at boundary 168", 10, 1, 168, 1, domain.ScoreboardVisiblePublic, false},
		{"verify TTL over boundary", 10, 1, 169, 1, domain.ScoreboardVisiblePublic, true},
		{"reset TTL at boundary 168", 10, 1, 24, 168, domain.ScoreboardVisiblePublic, false},
		{"reset TTL over boundary", 10, 1, 24, 169, domain.ScoreboardVisiblePublic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := newSettingsTestDeps(t)
			uc, redisClient := d.createSettingsUseCase(t)

			settings := newTestAppSettingsWithValues(tt.submitLimit, tt.submitDur, tt.verifyTTL, tt.resetTTL, tt.visibility)

			if !tt.wantErr {
				d.SettingsRepo.On("GetForUpdate", mock.Anything).Return(settings, nil)
				d.SettingsRepo.On("UpdateIfCurrent", mock.Anything, settings).Return(nil)
				redisClient.ExpectDel(cache.KeyAppSettings).SetVal(1)
				d.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
			}

			err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.NoError(t, redisClient.ExpectationsWereMet())
		})
	}
}
