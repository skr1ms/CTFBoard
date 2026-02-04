package competition

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	challengeMocks "github.com/skr1ms/CTFBoard/internal/usecase/challenge/mocks"
	"github.com/skr1ms/CTFBoard/internal/usecase/competition/mocks"
	teamMocks "github.com/skr1ms/CTFBoard/internal/usecase/team/mocks"
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
	txRepo          *mocks.MockTxRepository
	statsRepo       *mocks.MockStatisticsRepository
	appSettingsRepo *mocks.MockAppSettingsRepository
	hintRepo        *challengeMocks.MockHintRepository
	teamRepo        *teamMocks.MockTeamRepository
	awardRepo       *teamMocks.MockAwardRepository
	backupRepo      *mocks.MockBackupRepository
	logger          *mocks.MockLogger
	bracketRepo     *mocks.MockBracketRepository
	configRepo      *mocks.MockConfigRepository
	ratingRepo      *mocks.MockRatingRepository
	submissionRepo  *mocks.MockSubmissionRepository
}

func NewCompetitionTestHelper(t *testing.T) *CompetitionTestHelper {
	t.Helper()

	l := mocks.NewMockLogger(t)
	l.On("Info", mock.Anything, mock.Anything).Maybe()
	l.On("Warn", mock.Anything, mock.Anything).Maybe()
	l.On("Error", mock.Anything, mock.Anything).Maybe()
	l.On("Debug", mock.Anything, mock.Anything).Maybe()

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
			appSettingsRepo: mocks.NewMockAppSettingsRepository(t),
			hintRepo:        challengeMocks.NewMockHintRepository(t),
			teamRepo:        teamMocks.NewMockTeamRepository(t),
			awardRepo:       teamMocks.NewMockAwardRepository(t),
			backupRepo:      mocks.NewMockBackupRepository(t),
			logger:          l,
			bracketRepo:     mocks.NewMockBracketRepository(t),
			configRepo:      mocks.NewMockConfigRepository(t),
			ratingRepo:      mocks.NewMockRatingRepository(t),
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
	return NewCompetitionUseCase(h.deps.competitionRepo, h.deps.auditLogRepo, client), redis
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
