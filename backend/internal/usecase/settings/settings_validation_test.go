package settings

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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
