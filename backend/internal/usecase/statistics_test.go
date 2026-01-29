package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStatisticsUseCase_GetGeneralStats_Success(t *testing.T) {
	statsRepo := mocks.NewMockStatisticsRepository(t)
	db, redisClient := redismock.NewClientMock()

	stats := &entity.GeneralStats{
		UserCount:      100,
		TeamCount:      20,
		ChallengeCount: 15,
		SolveCount:     50,
	}

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	statsRepo.On("GetGeneralStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:general", `.*`, 5*time.Minute).SetVal("OK")

	uc := NewStatisticsUseCase(statsRepo, db)

	result, err := uc.GetGeneralStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, stats.UserCount, result.UserCount)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Cached(t *testing.T) {
	statsRepo := mocks.NewMockStatisticsRepository(t)
	db, redisClient := redismock.NewClientMock()

	stats := &entity.GeneralStats{
		UserCount: 100,
	}
	bytes, _ := json.Marshal(stats)

	redisClient.ExpectGet("stats:general").SetVal(string(bytes))

	uc := NewStatisticsUseCase(statsRepo, db)

	result, err := uc.GetGeneralStats(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, 100, result.UserCount)
	statsRepo.AssertNotCalled(t, "GetGeneralStats", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetGeneralStats_Error(t *testing.T) {
	statsRepo := mocks.NewMockStatisticsRepository(t)
	db, redisClient := redismock.NewClientMock()

	redisClient.ExpectGet("stats:general").SetErr(redis.Nil)
	statsRepo.On("GetGeneralStats", mock.Anything).Return(nil, errors.New("db error"))

	uc := NewStatisticsUseCase(statsRepo, db)

	result, err := uc.GetGeneralStats(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetChallengeStats_Success(t *testing.T) {
	statsRepo := mocks.NewMockStatisticsRepository(t)
	db, redisClient := redismock.NewClientMock()

	stats := []*entity.ChallengeStats{
		{Id: uuid.New(), Title: "Chall 1", SolveCount: 10},
	}

	redisClient.ExpectGet("stats:challenges").SetErr(redis.Nil)
	statsRepo.On("GetChallengeStats", mock.Anything).Return(stats, nil)
	redisClient.Regexp().ExpectSet("stats:challenges", `.*`, 5*time.Minute).SetVal("OK")

	uc := NewStatisticsUseCase(statsRepo, db)

	result, err := uc.GetChallengeStats(context.Background())

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, "Chall 1", result[0].Title)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestStatisticsUseCase_GetScoreboardHistory_Success(t *testing.T) {
	statsRepo := mocks.NewMockStatisticsRepository(t)
	db, redisClient := redismock.NewClientMock()

	history := []*entity.ScoreboardHistoryEntry{
		{TeamId: uuid.New(), Points: 100, Timestamp: time.Now()},
	}

	redisClient.ExpectGet("stats:history:10").SetErr(redis.Nil)
	statsRepo.On("GetScoreboardHistory", mock.Anything, 10).Return(history, nil)
	redisClient.Regexp().ExpectSet("stats:history:10", `.*`, 30*time.Second).SetVal("OK")

	uc := NewStatisticsUseCase(statsRepo, db)

	result, err := uc.GetScoreboardHistory(context.Background(), 10)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}
