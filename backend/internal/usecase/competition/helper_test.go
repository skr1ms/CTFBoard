package competition

import (
	"archive/zip"
	"bytes"
	"encoding/json"
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
	"github.com/stretchr/testify/require"
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

func (h *CompetitionTestHelper) CreateSettingsUseCase() (*SettingsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSettingsUseCase(h.deps.appSettingsRepo, h.deps.auditLogRepo, client), redis
}

func (h *CompetitionTestHelper) NewAppSettings() *entity.AppSettings {
	h.t.Helper()
	return &entity.AppSettings{
		ID:                     1,
		AppName:                "CTFBoard",
		VerifyEmails:           true,
		FrontendURL:            "http://localhost:3000",
		CORSOrigins:            "http://localhost:3000",
		ResendEnabled:          false,
		ResendFromEmail:        "noreply@ctfboard.local",
		ResendFromName:         "CTFBoard",
		VerifyTTLHours:         24,
		ResetTTLHours:          1,
		SubmitLimitPerUser:     10,
		SubmitLimitDurationMin: 1,
		ScoreboardVisible:      entity.ScoreboardVisiblePublic,
		RegistrationOpen:       true,
		UpdatedAt:              time.Now(),
	}
}

func (h *CompetitionTestHelper) NewAppSettingsWithValues(
	submitLimit int,
	submitDuration int,
	verifyTTL int,
	resetTTL int,
	visibility string,
) *entity.AppSettings {
	h.t.Helper()
	s := h.NewAppSettings()
	s.SubmitLimitPerUser = submitLimit
	s.SubmitLimitDurationMin = submitDuration
	s.VerifyTTLHours = verifyTTL
	s.ResetTTLHours = resetTTL
	s.ScoreboardVisible = visibility
	return s
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

func (h *CompetitionTestHelper) CreateBackupUseCase() *BackupUseCase {
	h.t.Helper()
	return NewBackupUseCase(
		h.deps.competitionRepo,
		h.deps.challengeRepo,
		h.deps.hintRepo,
		h.deps.teamRepo,
		h.deps.userRepo,
		h.deps.awardRepo,
		h.deps.solveRepo,
		nil,
		h.deps.backupRepo,
		nil,
		h.deps.txRepo,
		h.deps.logger,
	)
}

func (h *CompetitionTestHelper) SetupBackupExportMocks(comp *entity.Competition, challenges []*repo.ChallengeWithSolved, challengeID uuid.UUID) {
	h.t.Helper()
	h.deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	h.deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil)).Return(challenges, nil)
	h.deps.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*entity.Hint{}, nil)
}

func (h *CompetitionTestHelper) NewMinimalBackupData() *entity.BackupData {
	h.t.Helper()
	comp := h.NewCompetition("CTF", "flexible", true)
	challengeID := uuid.New()
	return &entity.BackupData{
		Version:     entity.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: comp,
		Challenges: []entity.ChallengeExport{
			{
				Challenge: *h.NewChallenge(challengeID, "Chall", 100),
				Hints:     []entity.Hint{},
			},
		},
	}
}

func (h *CompetitionTestHelper) BuildBackupZip(data *entity.BackupData) ([]byte, int64) {
	h.t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	w, err := zw.Create("backup.json")
	require.NoError(h.t, err)
	require.NoError(h.t, json.NewEncoder(w).Encode(data))
	_ = zw.Close()

	b := buf.Bytes()
	return b, int64(len(b))
}
