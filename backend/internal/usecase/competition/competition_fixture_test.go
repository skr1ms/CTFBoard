package competition

import (
	"context"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/wahrwelt-kit/go-cachekit"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
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
		ID:              1,
		Name:            name,
		Mode:            domain.CompetitionMode(mode),
		AllowTeamSwitch: allowTeamSwitch,
		MinTeamSize:     1,
		MaxTeamSize:     10,
	}
}

func newTestCompetitionWithTimes(name string, startTime, endTime *time.Time) *domain.Competition {
	c := newTestCompetition(name, "teams_only", true)
	c.StartTime = startTime
	c.EndTime = endTime

	return c
}

func withDefaultTeamSizes(c *domain.Competition) *domain.Competition {
	if c.MinTeamSize == 0 {
		c.MinTeamSize = 1
	}

	if c.MaxTeamSize == 0 {
		c.MaxTeamSize = 10
	}

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

func newTestScoreboardEntry(teamID uuid.UUID, teamName string, points int) *domain.ScoreboardEntry {
	return &domain.ScoreboardEntry{TeamID: teamID, TeamName: teamName, Points: points, SolvedAt: time.Now()}
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
