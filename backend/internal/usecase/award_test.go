package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAwardUseCase_Create(t *testing.T) {
	mockRepo := mocks.NewMockAwardRepository(t)
	mockRedis := mocks.NewMockRedisClient(t)
	uc := usecase.NewAwardUseCase(mockRepo, mockRedis)

	ctx := context.Background()
	teamID := uuid.New()
	adminID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		mockRepo.On("Create", ctx, mock.MatchedBy(func(a *entity.Award) bool {
			return a.TeamId == teamID && a.Value == 100 && a.Description == "Bonus" && *a.CreatedBy == adminID
		})).Return(nil).Once()

		mockRedis.On("Del", ctx, []string{"scoreboard"}).Return(redis.NewIntCmd(ctx)).Once()
		mockRedis.On("Del", ctx, []string{"scoreboard:frozen"}).Return(redis.NewIntCmd(ctx)).Once()

		award, err := uc.Create(ctx, teamID, 100, "Bonus", adminID)

		assert.NoError(t, err)
		assert.NotNil(t, award)
		assert.Equal(t, 100, award.Value)
		assert.Equal(t, adminID, *award.CreatedBy)
	})

	t.Run("ZeroValue", func(t *testing.T) {
		award, err := uc.Create(ctx, teamID, 0, "Zero", adminID)

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "value cannot be 0")
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo.On("Create", ctx, mock.Anything).Return(errors.New("db error")).Once()

		award, err := uc.Create(ctx, teamID, 50, "Error", adminID)

		assert.Error(t, err)
		assert.Nil(t, award)
		assert.Contains(t, err.Error(), "db error")
	})
}

func TestAwardUseCase_GetByTeamID(t *testing.T) {
	mockRepo := mocks.NewMockAwardRepository(t)
	mockRedis := mocks.NewMockRedisClient(t)
	uc := usecase.NewAwardUseCase(mockRepo, mockRedis)

	ctx := context.Background()
	teamID := uuid.New()

	t.Run("Success", func(t *testing.T) {
		expectedAwards := []*entity.Award{
			{Id: uuid.New(), TeamId: teamID, Value: 100, CreatedAt: time.Now()},
			{Id: uuid.New(), TeamId: teamID, Value: -50, CreatedAt: time.Now()},
		}

		mockRepo.On("GetByTeamID", ctx, teamID).Return(expectedAwards, nil).Once()

		awards, err := uc.GetByTeamID(ctx, teamID)

		assert.NoError(t, err)
		assert.Equal(t, len(expectedAwards), len(awards))
		assert.Equal(t, expectedAwards[0].Id, awards[0].Id)
	})

	t.Run("RepoError", func(t *testing.T) {
		mockRepo.On("GetByTeamID", ctx, teamID).Return(nil, errors.New("db error")).Once()

		awards, err := uc.GetByTeamID(ctx, teamID)

		assert.Error(t, err)
		assert.Nil(t, awards)
	})
}
