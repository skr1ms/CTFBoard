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
	"github.com/wahrwelt-kit/go-cachekit"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	challengeMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mock"
	compMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mock"
	teamMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mock"
)

type competitionTestDeps struct {
	competitionRepo *compMock.MockCompetitionRepository
	auditLogRepo    *compMock.MockAuditLogRepository
	solveRepo       *compMock.MockSolveRepository
	challengeRepo   *compMock.MockChallengeRepository
	userRepo        *compMock.MockUserRepository
	tm              *compMock.MockTransactionManager
	statsRepo       *compMock.MockStatisticsRepository
	SettingsRepo    *compMock.MockSettingsRepository
	hintRepo        *challengeMock.MockHintRepository
	teamRepo        *teamMock.MockTeamRepository
	awardRepo       *teamMock.MockAwardRepository
	logger          *logMock.MockLogger
	bracketRepo     *compMock.MockBracketRepository
	configRepo      *compMock.MockCompetitionParamRepo
	submissionRepo  *compMock.MockSubmissionRepository
}

func newCompetitionTestDeps(t *testing.T) *competitionTestDeps {
	t.Helper()
	l := logMock.NewMockLogger(t)
	l.On("Info", mock.Anything, mock.Anything).Maybe()
	l.On("Warn", mock.Anything, mock.Anything).Maybe()
	l.On("Error", mock.Anything, mock.Anything).Maybe()
	l.On("Debug", mock.Anything, mock.Anything).Maybe()
	l.On("WithError", mock.Anything).Return(l).Maybe()
	l.On("WithFields", mock.Anything).Return(l).Maybe()

	return &competitionTestDeps{
		competitionRepo: compMock.NewMockCompetitionRepository(t),
		auditLogRepo:    compMock.NewMockAuditLogRepository(t),
		solveRepo:       compMock.NewMockSolveRepository(t),
		challengeRepo:   compMock.NewMockChallengeRepository(t),
		userRepo:        compMock.NewMockUserRepository(t),
		tm:              compMock.NewMockTransactionManager(t),
		statsRepo:       compMock.NewMockStatisticsRepository(t),
		SettingsRepo:    compMock.NewMockSettingsRepository(t),
		hintRepo:        challengeMock.NewMockHintRepository(t),
		teamRepo:        teamMock.NewMockTeamRepository(t),
		awardRepo:       teamMock.NewMockAwardRepository(t),
		logger:          l,
		bracketRepo:     compMock.NewMockBracketRepository(t),
		configRepo:      compMock.NewMockCompetitionParamRepo(t),
		submissionRepo:  compMock.NewMockSubmissionRepository(t),
	}
}

func (d *competitionTestDeps) createCompetitionUseCase() (*CompetitionUseCase, redismock.ClientMock) {
	client, redis := redismock.NewClientMock()

	return NewCompetitionUseCase(CompetitionDeps{
		CompetitionRepo: d.competitionRepo, AuditLogRepo: d.auditLogRepo, TM: d.tm,
		Redis: &cachekit.RedisKeyValueStore{Client: client}, Logger: d.logger,
	}), redis
}

func (d *competitionTestDeps) createSubmissionUseCase() *SubmissionUseCase {
	return NewSubmissionUseCase(SubmissionDeps{SubmissionRepo: d.submissionRepo})
}

func (d *competitionTestDeps) createSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	client, redis := redismock.NewClientMock()

	return NewSolveUseCase(SolveDeps{
		SolveRepo: d.solveRepo, ChallengeRepo: d.challengeRepo, CompetitionRepo: d.competitionRepo,
		UserRepo: d.userRepo, TeamRepo: d.teamRepo, TM: d.tm, Cache: cachekit.New(client),
		ScoreboardCache: nil, ChallengeListCache: nil, Broadcaster: nil,
	}), redis
}

func (d *competitionTestDeps) createBracketUseCase() *BracketUseCase {
	return NewBracketUseCase(BracketDeps{BracketRepo: d.bracketRepo, TM: d.tm})
}

func (d *competitionTestDeps) createStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	client, mock := redismock.NewClientMock()

	return NewStatisticsUseCase(StatisticsDeps{StatsRepo: d.statsRepo, Cache: cachekit.New(client)}), mock
}

func (d *competitionTestDeps) createCompetitionParamUseCase() *CompetitionParamUseCase {
	return d.createCompetitionParamUseCaseWithCache(nil, nil)
}

func (d *competitionTestDeps) createCompetitionParamUseCaseWithCache(cache cachekit.KeyValueStore, pubsub cachekit.PubSubStore) *CompetitionParamUseCase {
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error { return fn(ctx) }).
		Maybe()

	return NewCompetitionParamUseCase(CompetitionParamDeps{
		Repo: d.configRepo, AuditLogRepo: d.auditLogRepo, TM: d.tm, Logger: d.logger,
		Cache: cache, PubSub: pubsub,
	})
}

func newTestCompetition(name, mode string, allowTeamSwitch bool) *domain.Competition {
	return &domain.Competition{
		ID: 1, Name: name, Mode: domain.CompetitionMode(mode), AllowTeamSwitch: allowTeamSwitch,
	}
}

func newTestCompetitionWithTimes(name string, startTime, endTime *time.Time) *domain.Competition {
	c := newTestCompetition(name, "flexible", true)
	c.StartTime = startTime
	c.EndTime = endTime

	return c
}

func newTestChallenge(id uuid.UUID, title string, points int) *domain.Challenge {
	return &domain.Challenge{ID: id, Title: title, Points: points, SolveCount: 0}
}

func newTestSolve(userID, teamID, challengeID uuid.UUID) *domain.Solve {
	return &domain.Solve{UserID: userID, TeamID: teamID, ChallengeID: challengeID}
}

func newTestUser(id uuid.UUID, teamID *uuid.UUID) *domain.User {
	return &domain.User{ID: id, TeamID: teamID}
}

func newTestScoreboardEntry(teamID uuid.UUID, teamName string, points int) *repo.ScoreboardEntry {
	return &repo.ScoreboardEntry{TeamID: teamID, TeamName: teamName, Points: points, SolvedAt: time.Now()}
}

func newTestSubmission(userID uuid.UUID, teamID *uuid.UUID, challengeID uuid.UUID, flag string, isCorrect bool) *domain.Submission {
	return &domain.Submission{
		ID: uuid.New(), UserID: userID, TeamID: teamID, ChallengeID: challengeID,
		SubmittedFlag: flag, IsCorrect: isCorrect,
	}
}

func newTestBracket(name, description string, isDefault bool) *domain.Bracket {
	return &domain.Bracket{ID: uuid.New(), Name: name, Description: description, IsDefault: isDefault}
}

func newTestCompetitionParam(key, value, description string, valueType domain.CompetitionParamValueType) *domain.CompetitionParam {
	return &domain.CompetitionParam{Key: key, Value: value, ValueType: valueType, Description: description}
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
	d.competitionRepo.On("Get", mock.Anything).Return(nil, apperr.ErrCompetitionNotFound)

	result, err := uc.Get(context.Background())

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, apperr.ErrCompetitionNotFound)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func Test_competitionCacheStale_StartTimeBoundary(t *testing.T) {
	t.Parallel()

	now := time.Now()
	startTimeJustPassed := now.Add(-10 * time.Second)
	comp := &domain.Competition{StartTime: &startTimeJustPassed}
	assert.True(t, competitionCacheStale(comp, now))

	startTimeLongAgo := now.Add(-boundaryInvalidateWindow - time.Second)
	compOld := &domain.Competition{StartTime: &startTimeLongAgo}
	assert.False(t, competitionCacheStale(compOld, now))
}

func TestCompetitionUseCase_Update_Success(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	comp := newTestCompetition("Updated CTF", "flexible", true)
	comp.MinTeamSize = 1
	comp.MaxTeamSize = 5

	currentNotStarted := newTestCompetitionWithTimes("Current", new(time.Now().Add(24*time.Hour)), nil)
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.ID == comp.ID &&
			c.Name == comp.Name &&
			c.Mode == comp.Mode &&
			c.AllowTeamSwitch == comp.AllowTeamSwitch &&
			c.MinTeamSize == comp.MinTeamSize &&
			c.MaxTeamSize == comp.MaxTeamSize
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.MatchedBy(func(a *domain.AuditLog) bool {
		return a.Action == domain.AuditActionUpdate && a.EntityType == domain.AuditEntityCompetition
	})).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.NoError(t, err)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func optionalsFromComp(c *domain.Competition) *usecase.CompetitionUpdateOptionals {
	o := &usecase.CompetitionUpdateOptionals{
		IsPaused:        new(c.IsPaused),
		IsPublic:        new(c.IsPublic),
		AllowTeamSwitch: new(c.AllowTeamSwitch),
	}
	if c.MinTeamSize != 0 || c.MaxTeamSize != 0 {
		minT, maxT := c.MinTeamSize, c.MaxTeamSize
		o.MinTeamSize, o.MaxTeamSize = &minT, &maxT
	}

	return o
}

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
	currentActive.Mode = domain.ModeFlexible
	currentActive.AllowTeamSwitch = true

	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := newTestCompetition("Updated CTF", "teams_only", true)
	err := uc.Update(context.Background(), comp, optionalsFromComp(comp), uuid.New(), "127.0.0.1")

	assert.Error(t, err)
	assert.ErrorIs(t, err, apperr.ErrCompetitionActiveCannotUpdate)
	assert.NoError(t, redisClient.ExpectationsWereMet())
}

func TestCompetitionUseCase_Update_PauseSetsTimestamp(t *testing.T) {
	t.Parallel()
	d := newCompetitionTestDeps(t)
	uc, redisClient := d.createCompetitionUseCase()

	startTime := time.Now().Add(-1 * time.Hour)
	endTime := time.Now().Add(23 * time.Hour)
	currentActive := newTestCompetitionWithTimes("CTF", &startTime, &endTime)
	currentActive.Mode = domain.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.IsPaused && c.PausedAt != nil && time.Since(*c.PausedAt) < time.Second
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: true, Mode: domain.ModeFlexible, AllowTeamSwitch: true}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
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

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil && c.EndTime.Equal(endTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
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

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	adminEndTime := now.Add(30 * time.Minute)
	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
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

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: nil, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
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

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return !c.IsPaused && c.PausedAt == nil &&
			c.EndTime != nil && c.EndTime.Equal(endTime) &&
			c.FreezeTime != nil && c.FreezeTime.Equal(freezeTime)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "CTF", IsPaused: false, Mode: domain.ModeFlexible}
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

	currentPaused := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime, FreezeTime: &freezeTime,
		IsPaused: true, PausedAt: &pausedAt,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	adminEndTime := now.Add(4 * time.Hour)

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
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

	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &adminEndTime, IsPaused: false, Mode: domain.ModeFlexible}
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
	currentNotStarted.Mode = domain.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	invalidEndTime := time.Now().Add(1 * time.Hour)
	comp := &domain.Competition{ID: 1, Name: "CTF", EndTime: &invalidEndTime, Mode: domain.ModeFlexible}
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
	currentNotStarted.Mode = domain.ModeFlexible
	currentNotStarted.FreezeTime = nil
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentNotStarted, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()

	comp := &domain.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &sameTime, FreezeTime: &sameTime, Mode: domain.ModeFlexible}
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
	currentEnded.Mode = domain.ModeFlexible
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentEnded, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.Name == "Updated"
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "Updated", Mode: domain.ModeFlexible}
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
	currentActive.Mode = domain.ModeFlexible
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
	assert.ErrorIs(t, err, apperr.ErrCompetitionActiveCannotUpdate)
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
	currentActive := &domain.Competition{
		ID: 1, Name: "CTF", Mode: domain.ModeFlexible,
		StartTime: &startTime, EndTime: &endTime,
	}
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentActive, nil).Once()
	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.EndTime != nil && c.EndTime.Before(now)
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	forceEndTime := now.Add(-1 * time.Minute)
	comp := &domain.Competition{ID: 1, Name: "CTF", StartTime: &startTime, EndTime: &forceEndTime, Mode: domain.ModeFlexible}
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
	currentPaused.Mode = domain.ModeFlexible
	currentPaused.IsPaused = true
	currentPaused.IsPublic = true
	currentPaused.AllowTeamSwitch = false
	d.competitionRepo.EXPECT().GetForUpdate(mock.Anything).Return(currentPaused, nil).Once()

	d.tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	d.competitionRepo.EXPECT().Update(mock.Anything, mock.MatchedBy(func(c *domain.Competition) bool {
		return c.Name == "Updated Name" && c.IsPaused && c.IsPublic && !c.AllowTeamSwitch
	})).Return(nil).Once()
	d.auditLogRepo.EXPECT().Create(mock.Anything, mock.Anything).Return(nil).Once()
	redisClient.ExpectDel(cache.KeyCompetition).SetVal(1)

	comp := &domain.Competition{ID: 1, Name: "Updated Name", Mode: domain.ModeFlexible}
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
