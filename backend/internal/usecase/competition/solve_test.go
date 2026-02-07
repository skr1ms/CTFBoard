package competition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/skr1ms/CTFBoard/internal/entity"
	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSolveUseCase_Create(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck
	})
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, solve).Return(nil)
	deps.txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_AlreadySolved(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(entityError.ErrAlreadySolved).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck
	})
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	existingSolve := &entity.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
		SolvedAt:    time.Now(),
	}
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrAlreadySolved))
}

func TestSolveUseCase_Create_CreateError(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)
	expectedError := assert.AnError

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedError).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck
	})
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, solve).Return(expectedError)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
}

func TestSolveUseCase_Create_AutoDetectTeam(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	userID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(userID, uuid.Nil, challengeID)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck
	})
	user := h.NewUser(userID, &teamID)
	deps.txRepo.On("LockUserTx", mock.Anything, mock.Anything, userID).Return(nil)
	deps.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.txRepo.On("GetChallengeByIDTx", mock.Anything, mock.Anything, challengeID).Return(challenge, nil)
	deps.txRepo.On("GetSolveByTeamAndChallengeTx", mock.Anything, mock.Anything, teamID, challengeID).Return(nil, nil)
	deps.txRepo.On("CreateSolveTx", mock.Anything, mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.TeamID == teamID
	})).Return(nil)
	deps.txRepo.On("IncrementChallengeSolveCountTx", mock.Anything, mock.Anything, challengeID).Return(1, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_NoTeamError(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	userID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(userID, uuid.Nil, challengeID)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(entityError.ErrNoTeamSelected).Run(func(args mock.Arguments) {
		fn, ok := args.Get(1).(func(context.Context, repo.Transaction) error)
		if !ok {
			return
		}
		_ = fn(context.Background(), nil) //nolint:errcheck
	})
	user := h.NewUser(userID, nil)
	deps.txRepo.On("LockUserTx", mock.Anything, mock.Anything, userID).Return(nil)
	deps.userRepo.On("GetByID", mock.Anything, userID).Return(user, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, entityError.ErrNoTeamSelected))
}

func TestSolveUseCase_GetScoreboard_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	entries := []*repo.ScoreboardEntry{
		h.NewScoreboardEntry(uuid.New(), "Team1", 500),
		h.NewScoreboardEntry(uuid.New(), "Team2", 300),
	}

	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)
	deps.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil)).Return(entries, nil)
	redisClient.Regexp().ExpectSet(cache.KeyScoreboard, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.GetScoreboard(context.Background(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[0].TeamName, result[0].TeamName)
	assert.Equal(t, entries[0].Points, result[0].Points)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Frozen(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	freezeTime := time.Now().Add(-1 * time.Hour)
	comp := h.NewCompetition("Test", "flexible", true)
	comp.FreezeTime = &freezeTime
	entries := []*repo.ScoreboardEntry{h.NewScoreboardEntry(uuid.New(), "Team1", 500)}

	redisClient.ExpectGet(cache.KeyScoreboardFrozen).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	deps.solveRepo.On("GetScoreboardByBracketFrozen", mock.Anything, freezeTime, (*uuid.UUID)(nil)).Return(entries, nil)
	redisClient.Regexp().ExpectSet(cache.KeyScoreboardFrozen, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.GetScoreboard(context.Background(), nil)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	expectedError := assert.AnError
	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, entityError.ErrCompetitionNotFound)
	deps.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil)).Return(nil, expectedError)

	result, err := uc.GetScoreboard(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetFirstBlood_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	challengeID := uuid.New()
	entry := &repo.FirstBloodEntry{
		UserID:   uuid.New(),
		Username: "firstsolver",
		TeamID:   uuid.New(),
		TeamName: "FirstTeam",
		SolvedAt: time.Now(),
	}

	deps.solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(entry, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, entry.Username, result.Username)
	assert.Equal(t, entry.TeamName, result.TeamName)
}

func TestSolveUseCase_GetFirstBlood_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	challengeID := uuid.New()
	deps.solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(nil, entityError.ErrSolveNotFound)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
