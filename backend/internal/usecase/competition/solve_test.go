package competition

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

func TestSolveUseCase_Create(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(uuid.New(), teamID, challengeID)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.userRepo.EXPECT().Lock(mock.Anything, solve.UserID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, solve.UserID).Return(&domain.User{ID: solve.UserID, IsBanned: false, TeamID: &teamID}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil).Once()
	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	challenge := newTestChallenge(challengeID, "Challenge", 100)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.solveRepo.EXPECT().Create(mock.Anything, solve).Return(nil)
	d.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_AlreadySolved(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(uuid.New(), teamID, challengeID)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		err := fn(ctx)
		if err != nil {
			return err
		}

		return apperr.ErrAlreadySolved
	})
	d.userRepo.EXPECT().Lock(mock.Anything, solve.UserID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, solve.UserID).Return(&domain.User{ID: solve.UserID, IsBanned: false, TeamID: &teamID}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil).Once()
	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	challenge := newTestChallenge(challengeID, "Challenge", 100)
	existingSolve := &domain.Solve{
		ID:          uuid.New(),
		TeamID:      teamID,
		ChallengeID: challengeID,
		SolvedAt:    time.Now(),
	}
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(existingSolve, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrAlreadySolved)
}

func TestSolveUseCase_Create_CreateError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	teamID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(uuid.New(), teamID, challengeID)
	expectedError := assert.AnError

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})
	d.userRepo.EXPECT().Lock(mock.Anything, solve.UserID).Return(nil).Once()
	d.userRepo.EXPECT().GetByID(mock.Anything, solve.UserID).Return(&domain.User{ID: solve.UserID, IsBanned: false, TeamID: &teamID}, nil).Once()
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(&domain.Team{ID: teamID, IsBanned: false}, nil).Once()
	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	challenge := newTestChallenge(challengeID, "Challenge", 100)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)
	d.solveRepo.EXPECT().Create(mock.Anything, solve).Return(expectedError)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
}

func TestSolveUseCase_Create_AutoDetectTeam(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	teamID := uuid.New()
	userID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(userID, uuid.Nil, challengeID)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	user := newTestUser(userID, &teamID)
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	team := &domain.Team{ID: teamID, IsBanned: false}
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(team, nil)
	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	challenge := newTestChallenge(challengeID, "Challenge", 100)
	d.challengeRepo.EXPECT().GetByID(mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.EXPECT().GetByTeamAndChallengeForUpdate(mock.Anything, teamID, challengeID).Return(nil, apperr.ErrSolveNotFound)
	d.solveRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(s *domain.Solve) bool {
		return s.TeamID == teamID
	})).Return(nil)
	d.challengeRepo.EXPECT().IncrementSolveCount(mock.Anything, challengeID).Return(1, nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	err := uc.Create(context.Background(), solve)

	assert.NoError(t, err)
}

func TestSolveUseCase_Create_NoTeamError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	userID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(userID, uuid.Nil, challengeID)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		err := fn(ctx)
		if err != nil {
			return err
		}

		return apperr.ErrNoTeamSelected
	})

	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	user := newTestUser(userID, nil)
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrNoTeamSelected)
}

func TestSolveUseCase_Create_TeamBanned_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	teamID := uuid.New()
	userID := uuid.New()
	challengeID := uuid.New()
	solve := newTestSolve(userID, uuid.Nil, challengeID)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	})

	d.competitionRepo.On("GetForUpdate", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound).Once()

	user := newTestUser(userID, &teamID)
	d.userRepo.EXPECT().Lock(mock.Anything, userID).Return(nil)
	d.userRepo.EXPECT().GetByID(mock.Anything, userID).Return(user, nil)

	bannedTeam := &domain.Team{ID: teamID, IsBanned: true}
	d.teamRepo.EXPECT().Lock(mock.Anything, teamID).Return(nil).Once()
	d.teamRepo.EXPECT().GetByID(mock.Anything, teamID).Return(bannedTeam, nil)

	err := uc.Create(context.Background(), solve)

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrTeamBanned)
}

func TestSolveUseCase_GetScoreboard_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createSolveUseCase()

	entries := []*repo.ScoreboardEntry{
		newTestScoreboardEntry(uuid.New(), "Team1", 500),
		newTestScoreboardEntry(uuid.New(), "Team2", 300),
	}

	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound)
	d.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil), (*time.Time)(nil)).Return(entries, nil)
	redisClient.Regexp().ExpectSet(cache.KeyScoreboard, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.GetScoreboard(context.Background(), nil, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 2)
	assert.Equal(t, entries[0].TeamName, result[0].TeamName)
	assert.Equal(t, entries[0].Points, result[0].Points)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Frozen(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createSolveUseCase()

	freezeTime := time.Now().Add(-1 * time.Hour)
	startTime := time.Now().Add(-2 * time.Hour)
	comp := newTestCompetition("Test", "teams_only", true)
	comp.StartTime = &startTime
	comp.FreezeTime = &freezeTime
	entries := []*repo.ScoreboardEntry{newTestScoreboardEntry(uuid.New(), "Team1", 500)}

	frozenKey := cache.KeyScoreboardFrozenAt(freezeTime.Unix())
	redisClient.ExpectGet(frozenKey).SetErr(redis.Nil)
	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	d.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil), &freezeTime).Return(entries, nil)
	redisClient.Regexp().ExpectSet(frozenKey, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.GetScoreboard(context.Background(), nil, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result, 1)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetScoreboard_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createSolveUseCase()

	expectedError := assert.AnError

	redisClient.ExpectGet(cache.KeyScoreboard).SetErr(redis.Nil)
	d.tm.EXPECT().ReadOnly(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound)
	d.solveRepo.On("GetScoreboardByBracket", mock.Anything, (*uuid.UUID)(nil), (*time.Time)(nil)).Return(nil, expectedError)

	result, err := uc.GetScoreboard(context.Background(), nil, false)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestSolveUseCase_GetFirstBlood_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	challengeID := uuid.New()
	entry := &repo.FirstBloodEntry{
		UserID:   uuid.New(),
		Username: "firstsolver",
		TeamID:   uuid.New(),
		TeamName: "FirstTeam",
		SolvedAt: time.Now(),
	}

	activeComp := newTestCompetitionWithTimes("CTF", new(time.Now().Add(-time.Hour)), new(time.Now().Add(time.Hour)))
	d.competitionRepo.On("Get", mock.Anything).Return(activeComp, nil).Maybe()

	challenge := newTestChallenge(challengeID, "Test", 100)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetFirstBlood", mock.Anything, challengeID, (*time.Time)(nil)).Return(entry, nil)

	result, err := uc.GetFirstBlood(context.Background(), challengeID, false)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, entry.Username, result.Username)
	assert.Equal(t, entry.TeamName, result.TeamName)
}

func TestSolveUseCase_GetFirstBlood_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, _ := d.createSolveUseCase()

	activeComp := newTestCompetitionWithTimes("CTF", new(time.Now().Add(-time.Hour)), new(time.Now().Add(time.Hour)))
	d.competitionRepo.On("Get", mock.Anything).Return(activeComp, nil).Maybe()

	challengeID := uuid.New()
	challenge := newTestChallenge(challengeID, "Test", 100)
	d.challengeRepo.On("GetByID", mock.Anything, challengeID).Return(challenge, nil)
	d.solveRepo.On("GetFirstBlood", mock.Anything, challengeID, (*time.Time)(nil)).Return(nil, apperr.ErrSolveNotFound)

	result, err := uc.GetFirstBlood(context.Background(), challengeID, false)

	assert.Error(t, err)
	assert.Nil(t, result)
}
