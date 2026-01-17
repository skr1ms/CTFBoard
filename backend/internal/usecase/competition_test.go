package usecase

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestCompetitionUseCase_Get(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	comp := &entity.Competition{
		Id:   1,
		Name: "Test CTF",
	}

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "competition").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.On("Set", mock.Anything, "competition", mock.Anything, 5*time.Second).Return(redis.NewStatusCmd(context.Background()))

	uc := NewCompetitionUseCase(competitionRepo, redisClient)

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
}

func TestCompetitionUseCase_Get_Cached(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	comp := &entity.Competition{
		Id:   1,
		Name: "Test CTF",
	}
	bytes, _ := json.Marshal(comp)

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetVal(string(bytes))
	redisClient.On("Get", mock.Anything, "competition").Return(cmd)

	uc := NewCompetitionUseCase(competitionRepo, redisClient)

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	competitionRepo.AssertNotCalled(t, "Get", mock.Anything)
}

func TestCompetitionUseCase_Get_NotFound(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "competition").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	uc := NewCompetitionUseCase(competitionRepo, redisClient)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, entityError.ErrCompetitionNotFound)
}

func TestCompetitionUseCase_Update(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	comp := &entity.Competition{
		Id:   1,
		Name: "Updated CTF",
	}

	competitionRepo.On("Update", mock.Anything, comp).Return(nil)
	redisClient.On("Del", mock.Anything, []string{"competition"}).Return(redis.NewIntCmd(context.Background()))

	uc := NewCompetitionUseCase(competitionRepo, redisClient)

	err := uc.Update(context.Background(), comp)

	assert.NoError(t, err)
}

func TestCompetitionUseCase_GetStatus(t *testing.T) {
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	startTime := time.Now().Add(-1 * time.Hour)
	comp := &entity.Competition{
		Id:        1,
		Name:      "Test CTF",
		StartTime: &startTime,
	}

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "competition").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	redisClient.On("Set", mock.Anything, "competition", mock.Anything, 5*time.Second).Return(redis.NewStatusCmd(context.Background()))

	uc := NewCompetitionUseCase(competitionRepo, redisClient)

	status, err := uc.GetStatus(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, entity.CompetitionStatusActive, status)
}
