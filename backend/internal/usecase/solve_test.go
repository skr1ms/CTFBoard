package usecase

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	db, redisClient := redismock.NewClientMock()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := &entity.Solve{
		UserId:      uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	redisClient.ExpectDel("scoreboard").SetVal(1)

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, pgx.Tx) error)
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

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_Create_AlreadySolved(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := &entity.Solve{
		UserId:      uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(entityError.ErrAlreadySolved).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, pgx.Tx) error)
		_ = fn(context.Background(), nil)
	})

	challenge := &entity.Challenge{
		Id:     challengeID,
		Title:  "Challenge",
		Points: 100,
	}
	existingSolve := &entity.Solve{
		Id:          uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
		SolvedAt:    time.Now(),
	}

	txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
}

func TestSolveUseCase_Create_CreateError(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := &entity.Solve{
		UserId:      uuid.New(),
		TeamId:      teamID,
		ChallengeId: challengeID,
	}
	expectedError := assert.AnError

	txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		fn := args.Get(1).(func(context.Context, pgx.Tx) error)
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

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
}

// GetScoreboard Tests

func TestSolveUseCase_GetScoreboard(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	entries := []*repo.ScoreboardEntry{
		{
			TeamId:   uuid.New(),
			TeamName: "Team1",
			Points:   500,
			SolvedAt: time.Now(),
		},
		{
			TeamId:   uuid.New(),
			TeamName: "Team2",
			Points:   300,
			SolvedAt: time.Now(),
		},
	}

	redisClient.ExpectGet("scoreboard").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	solveRepo.On("GetScoreboard", mock.Anything).Return(entries, nil)

	redisClient.Regexp().ExpectSet("scoreboard", `.*`, 15*time.Second).SetVal("OK")

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[0].TeamName, result[0].TeamName)
	assert.Equal(t, entries[0].Points, result[0].Points)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Frozen(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	freezeTime := time.Now().Add(-1 * time.Hour)
	comp := &entity.Competition{
		FreezeTime: &freezeTime,
	}

	entries := []*repo.ScoreboardEntry{
		{
			TeamId:   uuid.New(),
			TeamName: "Team1",
			Points:   500,
			SolvedAt: time.Now(),
		},
	}

	redisClient.ExpectGet("scoreboard:frozen").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(comp, nil)

	solveRepo.On("GetScoreboardFrozen", mock.Anything, freezeTime).Return(entries, nil)

	redisClient.Regexp().ExpectSet("scoreboard:frozen", `.*`, 15*time.Second).SetVal("OK")

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Error(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, redisClient := redismock.NewClientMock()

	expectedError := assert.AnError

	redisClient.ExpectGet("scoreboard").SetErr(redis.Nil)

	competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)

	solveRepo.On("GetScoreboard", mock.Anything).Return(nil, expectedError)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	result, err := uc.GetScoreboard(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

// GetFirstBlood Tests

func TestSolveUseCase_GetFirstBlood(t *testing.T) {
	solveRepo := mocks.NewMockSolveRepository(t)
	challengeRepo := mocks.NewMockChallengeRepository(t)
	competitionRepo := mocks.NewMockCompetitionRepository(t)
	txRepo := mocks.NewMockTxRepository(t)
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()
	entry := &repo.FirstBloodEntry{
		UserId:   uuid.New(),
		Username: "firstsolver",
		TeamId:   uuid.New(),
		TeamName: "FirstTeam",
		SolvedAt: time.Now(),
	}

	solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(entry, nil)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

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
	db, _ := redismock.NewClientMock()

	challengeID := uuid.New()

	solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(nil, entityError.ErrSolveNotFound)

	uc := NewSolveUseCase(solveRepo, challengeRepo, competitionRepo, txRepo, db, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
