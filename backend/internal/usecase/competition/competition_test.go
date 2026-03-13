package competition

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	challengeMocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
	teamMocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type competitionTestDeps struct {
	competitionRepo *mocks.MockCompetitionRepository
	auditLogRepo    *mocks.MockAuditLogRepository
	solveRepo       *mocks.MockSolveRepository
	challengeRepo   *mocks.MockChallengeRepository
	userRepo        *mocks.MockUserRepository
	tm              *mocks.MockTransactionManager
	statsRepo       *mocks.MockStatisticsRepository
	SettingsRepo    *mocks.MockSettingsRepository
	hintRepo        *challengeMocks.MockHintRepository
	teamRepo        *teamMocks.MockTeamRepository
	awardRepo       *teamMocks.MockAwardRepository
	logger          *mocks.MockLogger
	bracketRepo     *mocks.MockBracketRepository
	configRepo      *mocks.MockCompetitionParamRepo
	submissionRepo  *mocks.MockSubmissionRepository
}

func newCompetitionTestDeps(t *testing.T) *competitionTestDeps {
	t.Helper()
	l := mocks.NewMockLogger(t)
	l.On("Info", mock.Anything, mock.Anything).Maybe()
	l.On("Warn", mock.Anything, mock.Anything).Maybe()
	l.On("Error", mock.Anything, mock.Anything).Maybe()
	l.On("Debug", mock.Anything, mock.Anything).Maybe()
	l.On("WithError", mock.Anything).Return(l).Maybe()
	l.On("WithFields", mock.Anything).Return(l).Maybe()
	return &competitionTestDeps{
		competitionRepo: mocks.NewMockCompetitionRepository(t),
		auditLogRepo:    mocks.NewMockAuditLogRepository(t),
		solveRepo:       mocks.NewMockSolveRepository(t),
		challengeRepo:   mocks.NewMockChallengeRepository(t),
		userRepo:        mocks.NewMockUserRepository(t),
		tm:              mocks.NewMockTransactionManager(t),
		statsRepo:       mocks.NewMockStatisticsRepository(t),
		SettingsRepo:    mocks.NewMockSettingsRepository(t),
		hintRepo:        challengeMocks.NewMockHintRepository(t),
		teamRepo:        teamMocks.NewMockTeamRepository(t),
		awardRepo:       teamMocks.NewMockAwardRepository(t),
		logger:          l,
		bracketRepo:     mocks.NewMockBracketRepository(t),
		configRepo:      mocks.NewMockCompetitionParamRepo(t),
		submissionRepo:  mocks.NewMockSubmissionRepository(t),
	}
}

func (d *competitionTestDeps) createCompetitionUseCase() (*CompetitionUseCase, redismock.ClientMock) {
	client, redis := redismock.NewClientMock()
	return NewCompetitionUseCase(CompetitionDeps{
		CompetitionRepo: d.competitionRepo, AuditLogRepo: d.auditLogRepo, TM: d.tm,
		Redis: &cache.RedisKeyValueStore{Client: client}, Logger: d.logger,
	}), redis
}

func (d *competitionTestDeps) createSubmissionUseCase() *SubmissionUseCase {
	return NewSubmissionUseCase(SubmissionDeps{SubmissionRepo: d.submissionRepo})
}

func (d *competitionTestDeps) createSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	client, redis := redismock.NewClientMock()
	return NewSolveUseCase(SolveDeps{
		SolveRepo: d.solveRepo, ChallengeRepo: d.challengeRepo, CompetitionRepo: d.competitionRepo,
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, TM: d.tm, Cache: cache.New(client),
		ScoreboardCache: nil, ChallengeListCache: nil, Broadcaster: nil,
	}), redis
}

func (d *competitionTestDeps) createBracketUseCase() *BracketUseCase {
	return NewBracketUseCase(BracketDeps{BracketRepo: d.bracketRepo, TM: d.tm})
}

func (d *competitionTestDeps) createStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	client, mock := redismock.NewClientMock()
	return NewStatisticsUseCase(StatisticsDeps{StatsRepo: d.statsRepo, Cache: cache.New(client)}), mock
}

func (d *competitionTestDeps) createCompetitionParamUseCase() *CompetitionParamUseCase {
	return d.createCompetitionParamUseCaseWithCache(nil, nil)
}

func (d *competitionTestDeps) createCompetitionParamUseCaseWithCache(cache cache.KeyValueStore, pubsub cache.PubSubStore) *CompetitionParamUseCase {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).
		Maybe()
	return NewCompetitionParamUseCase(CompetitionParamDeps{
		Repo: d.configRepo, AuditLogRepo: d.auditLogRepo, TM: d.tm, Logger: d.logger,
		Cache: cache, PubSub: pubsub,
	})
}

func newTestCompetition(name, mode string, allowTeamSwitch bool) *entity.Competition {
	return &entity.Competition{
		ID: 1, Name: name, Mode: entity.CompetitionMode(mode), AllowTeamSwitch: allowTeamSwitch,
	}
}

func newTestCompetitionWithTimes(name string, startTime, endTime *time.Time) *entity.Competition {
	c := newTestCompetition(name, "flexible", true)
	c.StartTime = startTime
	c.EndTime = endTime
	return c
}

func newTestChallenge(id uuid.UUID, title string, points int) *entity.Challenge {
	return &entity.Challenge{ID: id, Title: title, Points: points, SolveCount: 0}
}

func newTestSolve(userID, teamID, challengeID uuid.UUID) *entity.Solve {
	return &entity.Solve{UserID: userID, TeamID: teamID, ChallengeID: challengeID}
}

func newTestUser(id uuid.UUID, teamID *uuid.UUID) *entity.User {
	return &entity.User{ID: id, TeamID: teamID}
}

func newTestScoreboardEntry(teamID uuid.UUID, teamName string, points int) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{TeamID: teamID, TeamName: teamName, Points: points, SolvedAt: time.Now()}
}

func newTestSubmission(userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, flag string, isCorrect bool) *entity.Submission {
	return &entity.Submission{
		ID: uuid.New(), UserID: userID, TeamID: teamID, ChallengeID: challengeID,
		SubmittedFlag: flag, IsCorrect: isCorrect,
	}
}

func newTestBracket(name, description string, isDefault bool) *entity.Bracket {
	return &entity.Bracket{ID: uuid.New(), Name: name, Description: description, IsDefault: isDefault}
}

func newTestCompetitionParam(key, value, description string, valueType entity.CompetitionParamValueType) *entity.CompetitionParam {
	return &entity.CompetitionParam{Key: key, Value: value, ValueType: valueType, Description: description}
}

func TestCompetitionUseCase_Get_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Test CTF", "flexible", true)

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	redisClient.Regexp().ExpectSet(cache.KeyCompetition, `.*`, 15*time.Second).SetVal("OK")

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	assert.Equal(t, comp.Mode, result.Mode)
	assert.Equal(t, comp.AllowTeamSwitch, result.AllowTeamSwitch)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_Cached_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Test CTF", "flexible", true)
	bytes, err := json.Marshal(comp)
	require.NoError(t, err)

	redisClient.ExpectGet(cache.KeyCompetition).SetVal(string(bytes))

	result, err := uc.Get(context.Background())

	assert.NoError(t, err)
	assert.Equal(t, comp.Name, result.Name)
	d.competitionRepo.AssertNotCalled(t, "Get", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Get_NotFound_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	redisClient.ExpectGet(cache.KeyCompetition).SetErr(redis.Nil)
	d.competitionRepo.On("Get", mock.Anything).Return(nil, httperr.ErrCompetitionNotFound)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, httperr.ErrCompetitionNotFound)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func Test_competitionCacheStale_StartTimeBoundary(t *testing.T) {
	t.Parallel()
	now := time.Now()
	startTimeJustPassed := now.Add(-10 * time.Second)
	comp := &entity.Competition{StartTime: &startTimeJustPassed}
	assert.True(t, competitionCacheStale(comp, now))

	startTimeLongAgo := now.Add(-boundaryInvalidateWindow - time.Second)
	compOld := &entity.Competition{StartTime: &startTimeLongAgo}
	assert.False(t, competitionCacheStale(compOld, now))
}

func TestCompetitionUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Updated CTF", "flexible", true)
	comp.MinTeamSize = 1
	comp.MaxTeamSize = 5

	currentNotStarted := newTestCompetitionWithTimes("Current", timePtr(time.Now().Add(24*time.Hour)), nil)
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.ID == comp.ID &&
			c.Name == comp.Name &&
			c.Mode == comp.Mode &&
			c.AllowTeamSwitch == comp.AllowTeamSwitch &&
			c.MinTeamSize == comp.MinTeamSize &&
			c.MaxTeamSize == comp.MaxTeamSize
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *entity.AuditLog) bool {
		return a.Action == entity.AuditActionUpdate && a.EntityType == entity.AuditEntityCompetition
	})).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func timePtr(t time.Time) *time.Time { return &t }

func optionalsFromComp(c *entity.Competition) *usecase.CompetitionUpdateOptionals {
	o := &usecase.CompetitionUpdateOptionals{
		IsPaused:        ptrBool(c.IsPaused),
		IsPublic:        ptrBool(c.IsPublic),
		AllowTeamSwitch: ptrBool(c.AllowTeamSwitch),
	}
	if c.MinTeamSize != 0 || c.MaxTeamSize != 0 {
		minT, maxT := c.MinTeamSize, c.MaxTeamSize
		o.MinTeamSize, o.MaxTeamSize = &minT, &maxT
	}
	return o
}

func ptrBool(b bool) *bool { return &b }

func TestCompetitionUseCase_Update_Error(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Updated CTF", "flexible", true)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).Return(errors.New("db error")).Once()

	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveCompetitionRejectsDangerousChanges(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	// Current competition is active
	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	currentActive := newTestCompetitionWithTimes("Active CTF", &startTime, &endTime)
	currentActive.Mode = entity.ModeFlexible
	currentActive.AllowTeamSwitch = true

	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := newTestCompetition("Updated CTF", "teams_only", true)
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrCompetitionActiveCannotUpdate)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_PauseSetsTimestamp(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(23 * time.Hour)
	currentActive := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentActive.Mode = entity.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.IsPaused && c.PausedAt != nil && time.Since(*c.PausedAt) < time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: true, Mode: entity.ModeFlexible, AllowTeamSwitch: true}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseShiftsEndTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(12 * time.Hour)
	pausedAt := now.Add(-2 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}
		shift := time.Since(pausedAt)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)
		return endShift > shift-time.Second && endShift < shift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWithEndTimeInPastForceEnds(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(-2 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil && c.EndTime.Equal(endTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWhenPausedBeforeEndTimeShiftsTimes(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(-1 * time.Hour)
	freezeTime := now.Add(-2 * time.Hour)
	pausedAt := now.Add(-3 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}
		shift := time.Since(pausedAt)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)
		return endShift > shift-time.Second && endShift < shift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseValidationRejectsFreezeAfterEnd(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	adminEndTime := now.Add(30 * time.Minute)
	comp := &entity.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "freeze_time must be before end_time") ||
		strings.Contains(err.Error(), "unpausing shifts freeze_time"),
		"error should mention freeze_time or unpause shift: %s", err.Error())
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseAfterPreStartPause_ClampsToStartTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(12 * time.Hour)
	pausedAt := now.Add(-3 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}
		effectiveShift := time.Since(startTime)
		endShift := c.EndTime.Sub(endTime)
		freezeShift := c.FreezeTime.Sub(freezeTime)
		return endShift > effectiveShift-time.Second && endShift < effectiveShift+time.Second &&
			c.FreezeTime != nil &&
			freezeShift > effectiveShift-time.Second && freezeShift < effectiveShift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseKeepsFreezeTimeUnchanged(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(14 * time.Hour)
	freezeTime := now.Add(-1 * time.Hour)
	pausedAt := now.Add(-30 * time.Minute)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_UnpauseWithNilEndTime_ShiftsFreezeTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-30 * time.Minute)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: nil, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		if c.IsPaused || c.PausedAt != nil || c.EndTime != nil {
			return false
		}
		shift := time.Since(pausedAt)
		freezeShift := c.FreezeTime.Sub(freezeTime)
		return c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseBeforeStartTime_NoShift(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(2 * time.Hour)
	endTime := now.Add(6 * time.Hour)
	freezeTime := now.Add(5 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil &&
			c.EndTime != nil && c.EndTime.Equal(endTime) &&
			c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_UnpauseWithChangedEndTime_StillShiftsFreezeTime(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-10 * time.Hour)
	endTime := now.Add(2 * time.Hour)
	freezeTime := now.Add(1 * time.Hour)
	pausedAt := now.Add(-1 * time.Hour)

	currentPaused := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	adminEndTime := now.Add(4 * time.Hour)
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		if c.IsPaused || c.PausedAt != nil {
			return false
		}
		if c.EndTime == nil || !c.EndTime.Equal(adminEndTime) {
			return false
		}
		shift := time.Since(pausedAt)
		freezeShift := c.FreezeTime.Sub(freezeTime)
		return c.FreezeTime != nil &&
			freezeShift > shift-time.Second && freezeShift < shift+time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_InvalidTimesAfterMergeReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(2 * time.Hour)
	endTime := time.Now().Add(24 * time.Hour)
	currentNotStarted := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentNotStarted.Mode = entity.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	invalidEndTime := time.Now().Add(1 * time.Hour)
	comp := &entity.Competition{ID: 1, Name: "CTF", EndTime: &invalidEndTime, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "end_time must be after start_time")
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_FreezeTimeEqualEndTime_ReturnsError(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(2 * time.Hour)
	sameTime := time.Now().Add(24 * time.Hour)
	currentNotStarted := newTestCompetitionWithTimes("CTF", &startTime, &sameTime)
	currentNotStarted.Mode = entity.ModeFlexible
	currentNotStarted.FreezeTime = nil
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := &entity.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &sameTime, FreezeTime: &sameTime, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "freeze_time must be before end_time")
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_EndedAllowsUpdate(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-48 * time.Hour)
	endTime := time.Now().Add(-1 * time.Hour)
	currentEnded := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentEnded.Mode = entity.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentEnded, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.Name == "Updated"
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "Updated", Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveRejectsTeamSizeChange(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(1 * time.Hour)
	currentActive := newTestCompetitionWithTimes("Active CTF", &startTime, &endTime)
	currentActive.Mode = entity.ModeFlexible
	currentActive.MinTeamSize = 1
	currentActive.MaxTeamSize = 5
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := newTestCompetition("Updated CTF", "flexible", true)
	comp.MinTeamSize = 2
	comp.MaxTeamSize = 6
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, httperr.ErrCompetitionActiveCannotUpdate)
	d.competitionRepo.AssertNotCalled(t, "Update", mock.Anything)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_ActiveAllowsEndTimeChange_ForceEnd(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	now := time.Now()
	startTime := now.Add(-1 * time.Hour)
	endTime := now.Add(23 * time.Hour)
	currentActive := &entity.Competition{
		ID: 1, Name: "CTF", Mode: entity.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.EndTime != nil && c.EndTime.Before(now)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	forceEndTime := now.Add(-1 * time.Minute)
	comp := &entity.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &forceEndTime, Mode: entity.ModeFlexible}
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

func TestCompetitionUseCase_Update_PartialUpdatePreservesBooleans(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(23 * time.Hour)
	currentPaused := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentPaused.Mode = entity.ModeFlexible
	currentPaused.IsPaused = true
	currentPaused.IsPublic = true
	currentPaused.AllowTeamSwitch = false
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *entity.Competition) bool {
		return c.Name == "Updated Name" && c.IsPaused && c.IsPublic && !c.AllowTeamSwitch
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &entity.Competition{ID: 1, Name: "Updated Name", Mode: entity.ModeFlexible}
	optionals := &usecase.CompetitionUpdateOptionals{}
	err := uc.Update(context.Background(), comp, optionals, uuid.New(), "127.0.0.1")
	assert.NoError(t, err)
}

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
	assert.Equal(t, entity.CompetitionStatusActive, status)
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
