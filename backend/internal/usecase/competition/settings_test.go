package competition

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	redisKeys "github.com/skr1ms/CTFBoard/pkg/redis"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSettingsUseCase_Get_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()

	redisClient.ExpectGet(redisKeys.KeyAppSettings).SetErr(redis.Nil)
	deps.appSettingsRepo.On("Get", mock.Anything).Return(settings, nil)
	redisClient.Regexp().ExpectSet(redisKeys.KeyAppSettings, `.*`, cacheTTL).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	assert.Equal(t, settings.SubmitLimitPerUser, result.SubmitLimitPerUser)
	assert.Equal(t, settings.ScoreboardVisible, result.ScoreboardVisible)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Get_Cached_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()
	bytes, err := json.Marshal(settings)
	require.NoError(t, err)

	redisClient.ExpectGet(redisKeys.KeyAppSettings).SetVal(string(bytes))

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	deps.appSettingsRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Get_RepoError(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	redisClient.ExpectGet(redisKeys.KeyAppSettings).SetErr(redis.Nil)
	deps.appSettingsRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "SettingsUseCase - Get")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()
	actorID := uuid.New()
	clientIP := "127.0.0.1"

	deps.appSettingsRepo.On("Update", mock.Anything, mock.MatchedBy(func(s *entity.AppSettings) bool {
		return s.ID == settings.ID && s.AppName == settings.AppName
	})).Return(nil)
	redisClient.ExpectDel(redisKeys.KeyAppSettings).SetVal(1)
	deps.auditLogRepo.On("Create", mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == entity.AuditActionUpdate &&
			a.EntityType == entity.AuditEntityAppSettings &&
			a.EntityID == "settings" &&
			a.IP == clientIP &&
			*a.UserID == actorID
	})).Return(nil)

	err := uc.Update(context.Background(), settings, actorID, clientIP)

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_RepoError(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()

	deps.appSettingsRepo.On("Update", mock.Anything, settings).Return(errors.New("db error"))

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SettingsUseCase - Update")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Update_AuditLogError(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()

	deps.appSettingsRepo.On("Update", mock.Anything, settings).Return(nil)
	redisClient.ExpectDel(redisKeys.KeyAppSettings).SetVal(1)
	deps.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(errors.New("audit error"))

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Create audit")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_SubmitLimitPerUser(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(0, 1, 24, 1, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit_limit_per_user must be >= 1")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_SubmitLimitDuration(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 0, 24, 1, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submit_limit_duration_min must be >= 1")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_VerifyTTL_TooLow(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 1, 0, 1, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_VerifyTTL_TooHigh(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 1, 200, 1, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "verify_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ResetTTL_TooLow(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 1, 24, 0, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ResetTTL_TooHigh(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 1, 24, 200, entity.ScoreboardVisiblePublic)

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "reset_ttl_hours must be between 1 and 168")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ScoreboardVisible_Invalid(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettingsWithValues(10, 1, 24, 1, "invalid")

	err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "scoreboard_visible must be public, hidden, or admins_only")
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_ScoreboardVisible_AllValid(t *testing.T) {
	validValues := []string{
		entity.ScoreboardVisiblePublic,
		entity.ScoreboardVisibleHidden,
		entity.ScoreboardVisibleAdminsOnly,
	}

	for _, visibility := range validValues {
		t.Run(visibility, func(t *testing.T) {
			h := NewCompetitionTestHelper(t)
			deps := h.Deps()
			uc, redisClient := h.CreateSettingsUseCase()

			settings := h.NewAppSettingsWithValues(10, 1, 24, 1, visibility)

			deps.appSettingsRepo.On("Update", mock.Anything, settings).Return(nil)
			redisClient.ExpectDel(redisKeys.KeyAppSettings).SetVal(1)
			deps.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(nil)

			err := uc.Update(context.Background(), settings, uuid.New(), "127.0.0.1")

			assert.NoError(t, err)
			assert.NoError(t, redisClient.ExpectationsWereMet())
		})
	}
}

func TestSettingsUseCase_Get_InvalidCachedJSON(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSettingsUseCase()

	settings := h.NewAppSettings()

	redisClient.ExpectGet(redisKeys.KeyAppSettings).SetVal("invalid json")
	deps.appSettingsRepo.On("Get", mock.Anything).Return(settings, nil)
	redisClient.Regexp().ExpectSet(redisKeys.KeyAppSettings, `.*`, cacheTTL).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, settings.AppName, result.AppName)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSettingsUseCase_Validate_BoundaryValues(t *testing.T) {
	tests := []struct {
		name        string
		submitLimit int
		submitDur   int
		verifyTTL   int
		resetTTL    int
		visibility  string
		wantErr     bool
	}{
		{"min valid values", 1, 1, 1, 1, entity.ScoreboardVisiblePublic, false},
		{"max valid TTL", 10, 1, 168, 168, entity.ScoreboardVisiblePublic, false},
		{"verify TTL at boundary 168", 10, 1, 168, 1, entity.ScoreboardVisiblePublic, false},
		{"verify TTL over boundary", 10, 1, 169, 1, entity.ScoreboardVisiblePublic, true},
		{"reset TTL at boundary 168", 10, 1, 24, 168, entity.ScoreboardVisiblePublic, false},
		{"reset TTL over boundary", 10, 1, 24, 169, entity.ScoreboardVisiblePublic, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := NewCompetitionTestHelper(t)
			deps := h.Deps()
			uc, redisClient := h.CreateSettingsUseCase()

			settings := h.NewAppSettingsWithValues(tt.submitLimit, tt.submitDur, tt.verifyTTL, tt.resetTTL, tt.visibility)

			if !tt.wantErr {
				deps.appSettingsRepo.On("Update", mock.Anything, settings).Return(nil)
				redisClient.ExpectDel(redisKeys.KeyAppSettings).SetVal(1)
				deps.auditLogRepo.On("Create", mock.Anything, mock.Anything).Return(nil)
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
