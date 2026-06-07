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
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

func TestCompetitionUseCase_GetStatus_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	comp := newTestCompetitionWithTimes("Test CTF", &startTime, nil)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.ExpectSet(cache.KeyCompetition, mock.Anything, 5*time.Second).SetVal("OK")

	status, err := uc.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, domain.CompetitionStatusActive, status)
}

func TestCompetitionUseCase_GetStatus_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, errors.New("db error"))

	status, err := uc.GetStatus(context.Background())

	assert.Error(t, err)
	assert.Empty(t, status)
}
