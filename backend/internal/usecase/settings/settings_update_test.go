package settings

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

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
