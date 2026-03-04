package competition

import (
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	challengeMocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/challenge/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/competition/mocks"
	teamMocks "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/team/mocks"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type CompetitionTestHelper struct {
	t    *testing.T
	deps *competitionTestDeps
}

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

func NewCompetitionTestHelper(t *testing.T) *CompetitionTestHelper {
	t.Helper()

	l := mocks.NewMockLogger(t)
	l.On("Info", mock.Anything, mock.Anything).Maybe()
	l.On("Warn", mock.Anything, mock.Anything).Maybe()
	l.On("Error", mock.Anything, mock.Anything).Maybe()
	l.On("Debug", mock.Anything, mock.Anything).Maybe()
	l.On("WithError", mock.Anything).Return(l).Maybe()
	l.On("WithFields", mock.Anything).Return(l).Maybe()

	tm := mocks.NewMockTransactionManager(t)

	return &CompetitionTestHelper{
		t: t,
		deps: &competitionTestDeps{
			competitionRepo: mocks.NewMockCompetitionRepository(t),
			auditLogRepo:    mocks.NewMockAuditLogRepository(t),
			solveRepo:       mocks.NewMockSolveRepository(t),
			challengeRepo:   mocks.NewMockChallengeRepository(t),
			userRepo:        mocks.NewMockUserRepository(t),
			tm:              tm,
			statsRepo:       mocks.NewMockStatisticsRepository(t),
			SettingsRepo:    mocks.NewMockSettingsRepository(t),
			hintRepo:        challengeMocks.NewMockHintRepository(t),
			teamRepo:        teamMocks.NewMockTeamRepository(t),
			awardRepo:       teamMocks.NewMockAwardRepository(t),
			logger:          l,
			bracketRepo:     mocks.NewMockBracketRepository(t),
			configRepo:      mocks.NewMockCompetitionParamRepo(t),
			submissionRepo:  mocks.NewMockSubmissionRepository(t),
		},
	}
}

func (h *CompetitionTestHelper) Deps() *competitionTestDeps {
	h.t.Helper()
	return h.deps
}

func (h *CompetitionTestHelper) CreateCompetitionUseCase() (*CompetitionUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewCompetitionUseCase(CompetitionDeps{
		CompetitionRepo: h.deps.competitionRepo,
		AuditLogRepo:    h.deps.auditLogRepo,
		TM:              h.deps.tm,
		Redis:           &cache.RedisKeyValueStore{Client: client},
		Logger:          h.deps.logger,
	}), redis
}

func (h *CompetitionTestHelper) NewCompetition(name, mode string, allowTeamSwitch bool) *entity.Competition {
	h.t.Helper()
	return &entity.Competition{
		ID:              1,
		Name:            name,
		Mode:            entity.CompetitionMode(mode),
		AllowTeamSwitch: allowTeamSwitch,
	}
}

func (h *CompetitionTestHelper) NewCompetitionWithTimes(name string, startTime, endTime *time.Time) *entity.Competition {
	h.t.Helper()
	c := h.NewCompetition(name, "flexible", true)
	c.StartTime = startTime
	c.EndTime = endTime
	return c
}

func (h *CompetitionTestHelper) NewChallenge(id uuid.UUID, title string, points int) *entity.Challenge {
	h.t.Helper()
	return &entity.Challenge{
		ID:         id,
		Title:      title,
		Points:     points,
		SolveCount: 0,
	}
}

func (h *CompetitionTestHelper) NewSolve(userID, teamID, challengeID uuid.UUID) *entity.Solve {
	h.t.Helper()
	return &entity.Solve{
		UserID:      userID,
		TeamID:      teamID,
		ChallengeID: challengeID,
	}
}

func (h *CompetitionTestHelper) NewUser(id uuid.UUID, teamID *uuid.UUID) *entity.User {
	h.t.Helper()
	return &entity.User{
		ID:     id,
		TeamID: teamID,
	}
}

func (h *CompetitionTestHelper) NewScoreboardEntry(teamID uuid.UUID, teamName string, points int) *repo.ScoreboardEntry {
	h.t.Helper()
	return &repo.ScoreboardEntry{
		TeamID:   teamID,
		TeamName: teamName,
		Points:   points,
		SolvedAt: time.Now(),
	}
}
