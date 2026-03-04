package competition

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestSolveUseCase_Create(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	deps.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil).Once()
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound).Once()
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.solveRepo.EXPECT().Create(mock.Anything, solve).Return(nil)
	deps.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_AlreadySolved(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return httperr.ErrAlreadySolved
	})
	deps.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil).Once()
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound).Once()
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	existingSolve := &entity.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
		SolvedAt:    time.Now(),
	}
	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrAlreadySolved))
}

func TestSolveUseCase_Create_CreateError(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(uuid.New(), teamID, challengeID)
	expectedError := assert.AnError

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	deps.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&entity.Team{ID: teamID, IsBanned: false}, nil).Once()
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound).Once()
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)
	deps.solveRepo.EXPECT().Create(mock.Anything, solve).Return(expectedError)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
}

func TestSolveUseCase_Create_AutoDetectTeam(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	userID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(userID, uuid.Nil, challengeID)

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	user := h.NewUser(userID, &teamID)
	deps.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	team := &entity.Team{ID: teamID, IsBanned: false}
	deps.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound).Once()
	challenge := h.NewChallenge(challengeID, "Challenge", 100)
	deps.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, httperr.ErrSolveNotFound)
	deps.solveRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *entity.Solve) bool {
		return s.TeamID == teamID
	})).Return(nil)
	deps.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_NoTeamError(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	userID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(userID, uuid.Nil, challengeID)

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		if err := fn(ctx); err != nil {
			return err
		}
		return httperr.ErrNoTeamSelected
	})
	user := h.NewUser(userID, nil)
	deps.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrNoTeamSelected))
}

func TestSolveUseCase_Create_TeamBanned_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	teamID := uuid.New()
	userID := uuid.New()
	challengeID := uuid.New()
	solve := h.NewSolve(userID, uuid.Nil, challengeID)

	deps.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	user := h.NewUser(userID, &teamID)
	deps.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	deps.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)
	bannedTeam := &entity.Team{ID: teamID, IsBanned: true}
	deps.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	deps.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(bannedTeam, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.True(t, errors.Is(err, httperr.ErrTeamBanned))
}

func TestSolveUseCase_GetScoreboard_Success(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	entries := []*repo.ScoreboardEntry{
		h.NewScoreboardEntry(uuid.New(), "Team1", 500),
		h.NewScoreboardEntry(uuid.New(), "Team2", 300),
	}

	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound)
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
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	freezeTime := time.Now().Add(-1 * time.Hour)
	startTime := time.Now().Add(-2 * time.Hour)
	comp := h.NewCompetition("Test", "flexible", true)
	comp.StartTime = &startTime
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
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, redisClient := h.CreateSolveUseCase()

	expectedError := assert.AnError
	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound)
	deps.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil)).Return(nil, expectedError)

	result, err := uc.GetScoreboard(context.Background(), nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetFirstBlood_Success(t *testing.T) {
	t.Parallel()
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

	challenge := h.NewChallenge(challengeID, "Test", 100)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(entry, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, entry.Username, result.Username)
	assert.Equal(t, entry.TeamName, result.TeamName)
}

func TestSolveUseCase_GetFirstBlood_Error(t *testing.T) {
	t.Parallel()
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc, _ := h.CreateSolveUseCase()

	challengeID := uuid.New()
	challenge := h.NewChallenge(challengeID, "Test", 100)
	deps.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	deps.solveRepo.On("GetFirstBlood", mock.Anything, challengeID).Return(nil, httperr.ErrSolveNotFound)

	result, err := uc.GetFirstBlood(context.Background(), challengeID)

	assert.Error(t, err)
	assert.Nil(t, result)
}
