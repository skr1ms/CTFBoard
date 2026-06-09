package settings

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
)

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
