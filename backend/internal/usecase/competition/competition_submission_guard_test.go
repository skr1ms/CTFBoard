package competition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
)

func TestCompetitionUseCase_IsSubmissionAllowed_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	comp := newTestCompetitionWithTimes("Test CTF", &startTime, &endTime)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.True(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_NotStarted_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(1 * time.Hour)
	comp := newTestCompetitionWithTimes("Test CTF", &startTime, nil)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Ended_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-2 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	comp := newTestCompetitionWithTimes("Test CTF", &startTime, &endTime)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.NoError(t, err)
	assert.False(t, allowed)
}

func TestCompetitionUseCase_IsSubmissionAllowed_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	allowed, err := uc.IsSubmissionAllowed(context.Background())

	assert.Error(t, err)
	assert.False(t, allowed)
}
