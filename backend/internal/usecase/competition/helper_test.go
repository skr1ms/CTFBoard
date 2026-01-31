package competition

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition/mocks"
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
	txRepo          *mocks.MockTxRepository
	statsRepo       *mocks.MockStatisticsRepository
}

func NewCompetitionTestHelper(t *testing.T) *CompetitionTestHelper {
	t.Helper()
	return &CompetitionTestHelper{
		t: t,
		deps: &competitionTestDeps{
			competitionRepo: mocks.NewMockCompetitionRepository(t),
			auditLogRepo:    mocks.NewMockAuditLogRepository(t),
			solveRepo:       mocks.NewMockSolveRepository(t),
			challengeRepo:   mocks.NewMockChallengeRepository(t),
			userRepo:        mocks.NewMockUserRepository(t),
			txRepo:          mocks.NewMockTxRepository(t),
			statsRepo:       mocks.NewMockStatisticsRepository(t),
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
	return NewCompetitionUseCase(h.deps.competitionRepo, h.deps.auditLogRepo, client), redis
}

func (h *CompetitionTestHelper) CreateSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSolveUseCase(
		h.deps.solveRepo,
		h.deps.challengeRepo,
		h.deps.competitionRepo,
		h.deps.userRepo,
		h.deps.txRepo,
		client,
		nil,
	), redis
}

func (h *CompetitionTestHelper) CreateStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewStatisticsUseCase(h.deps.statsRepo, client), redis
}

func (h *CompetitionTestHelper) NewCompetition(name, mode string, allowTeamSwitch bool) *entity.Competition {
	h.t.Helper()
	return &entity.Competition{
		ID:              1,
		Name:            name,
		Mode:            mode,
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
