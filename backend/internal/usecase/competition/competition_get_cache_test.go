package competition

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Test CTF", "teams_only", true)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.Regexp().ExpectSet(cache.KeyCompetition, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	assert.Equal(t, comp.Mode, result.Mode)
	assert.Equal(t, comp.AllowTeamSwitch, result.AllowTeamSwitch)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_Cached_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Test CTF", "teams_only", true)
	bytes, err := json.Marshal(comp)
	require.NoError(t, err)

	redisClient.ExpectGet(cache.KeyCompetition).SetVal(string(bytes))

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	d.competitionRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_NotFound_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrCompetitionNotFound)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func Test_competitionCacheStale_StartTimeBoundary(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startTimeJustPassed := now.Add(-10 * time.Second)
	comp := &domain.Competition{StartTime: &startTimeJustPassed}
	assert.True(t, competitionCacheStale(comp, now))

	startTimeLongAgo := now.Add(-boundaryInvalidateWindow - time.Second)
	compOld := &domain.Competition{StartTime: &startTimeLongAgo}
	assert.False(t, competitionCacheStale(compOld, now))
}
