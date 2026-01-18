package usecase

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Create Tests

func TestSolveUseCase_Create(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	challengeID := uuid.New().String()
	solve := &entity.Solve{
		UserId:      uuid.New().String(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	redisClient.On("Del", mock.Anything, []string{"scoreboard"}).Return(redis.NewIntCmd(context.Background()))

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, *sql.Tx) error)
		_ = fn(context.Background(), nil)
	})

	challenge := &entity.Challenge{
		Id:         challengeID,
		Title:      "Challenge",
		Points:     100,
		SolveCount: 0,
	}

	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, solve).Return(nil)
	txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_AlreadySolved(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	challengeID := uuid.New().String()
	solve := &entity.Solve{
		UserId:      uuid.New().String(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(entityError.ErrAlreadySolved).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, *sql.Tx) error)
		_ = fn(context.Background(), nil)
	})

	challenge := &entity.Challenge{
		Id:     challengeID,
		Title:  "Challenge",
		Points: 100,
	}
	existingSolve := &entity.Solve{
		Id:          uuid.New().String(),
		TeamId:      teamID,
		ChallengeId: challengeID,
		SolvedAt:    time.Now(),
	}

	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
}

func TestSolveUseCase_Create_CreateError(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	teamID := uuid.New().String()
	challengeID := uuid.New().String()
	solve := &entity.Solve{
		UserId:      uuid.New().String(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}
	expectedError := assert.AnError

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, *sql.Tx) error)
		_ = fn(context.Background(), nil)
	})

	challenge := &entity.Challenge{
		Id:     challengeID,
		Title:  "Challenge",
		Points: 100,
	}
	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, nil)
	txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, solve).Return(expectedError)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
}

// GetScoreboard Tests

func TestSolveUseCase_GetScoreboard(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	entries := []*repo.ScoreboardEntry{
		{
			TeamId:   uuid.New().String(),
			TeamName: "Team1",
			Points:   500,
			SolvedAt: time.Now(),
		},
		{
			TeamId:   uuid.New().String(),
			TeamName: "Team2",
			Points:   300,
			SolvedAt: time.Now(),
		},
	}

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "scoreboard").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	solveRepo.On("GetScoreboard", mock.Anything).Return(entries, nil)

	redisClient.On("Set", mock.Anything, "scoreboard", mock.Anything, 15*time.Second).Return(redis.NewStatusCmd(context.Background()))

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[0].TeamName, result[0].TeamName)
	assert.Equal(t, entries[0].Points, result[0].Points)
}

func TestSolveUseCase_GetScoreboard_Frozen(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	freezeTime := time.Now().Add(-1 * time.Hour)
	comp := &entity.Competition{
		FreezeTime: &freezeTime,
	}

	entries := []*repo.ScoreboardEntry{
		{
			TeamId:   uuid.New().String(),
			TeamName: "Team1",
			Points:   500,
			SolvedAt: time.Now(),
		},
	}

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "scoreboard:frozen").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	solveRepo.On("GetScoreboardFrozen", mock.Anything, freezeTime).Return(entries, nil)

	redisClient.On("Set", mock.Anything, "scoreboard:frozen", mock.Anything, 15*time.Second).Return(redis.NewStatusCmd(context.Background()))

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
}

func TestSolveUseCase_GetScoreboard_Error(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	expectedError := assert.AnError

	cmd := redis.NewStringCmd(context.Background())
	cmd.SetErr(redis.Nil)
	redisClient.On("Get", mock.Anything, "scoreboard").Return(cmd)

	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	solveRepo.On("GetScoreboard", mock.Anything).Return(nil, expectedError)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
}

// GetFirstBlood Tests

func TestSolveUseCase_GetFirstBlood(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New().String()
	entry := &repo.FirstBloodEntry{
		UserId:   uuid.New().String(),
		Username: "firstsolver",
		TeamId:   uuid.New().String(),
		TeamName: "FirstTeam",
		SolvedAt: time.Now(),
	}

	solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(entry, nil)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, entry.Username, result.Username)
	assert.Equal(t, entry.TeamName, result.TeamName)
}

func TestSolveUseCase_GetFirstBlood_NotFound(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	redisClient := mocks.NewMockRedisClient(t)

	challengeID := uuid.New().String()

	solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(nil, entityError.ErrSolveNotFound)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, redisClient, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
