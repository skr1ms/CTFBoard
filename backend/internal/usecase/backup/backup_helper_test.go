package backup

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type backupTestDeps struct {
	competitionRepo *mocks.MockCompetitionRepository
	challengeRepo   *mocks.MockChallengeRepository
	hintRepo        *mocks.MockHintRepository
	teamRepo        *mocks.MockTeamRepository
	userRepo        *mocks.MockUserRepository
	awardRepo       *mocks.MockAwardRepository
	solveRepo       *mocks.MockSolveRepository
	submissionRepo  *mocks.MockSubmissionRepository
	logger          *mocks.MockLogger
}

type BackupTestHelper struct {
	t    *testing.T
	deps *backupTestDeps
}

func (h *BackupTestHelper) CreateBackupUseCase() *BackupUseCase {
	h.t.Helper()
	return NewBackupUseCase(BackupDeps{
		CompetitionRepo: h.deps.competitionRepo,
		ChallengeRepo:   h.deps.challengeRepo,
		HintRepo:        h.deps.hintRepo,
		TeamRepo:        h.deps.teamRepo,
		UserRepo:        h.deps.userRepo,
		AwardRepo:       h.deps.awardRepo,
		SolveRepo:       h.deps.solveRepo,
		SubmissionRepo:  h.deps.submissionRepo,
		FileRepo:        nil,
		Storage:         nil,
		Logger:          h.deps.logger,
	})
}

func (h *BackupTestHelper) SetupBackupExportMocks(comp *entity.Competition, challenges []*repo.ChallengeWithSolved, challengeID uuid.UUID) {
	h.t.Helper()
	h.deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	h.deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(challenges, nil)
	h.deps.hintRepo.On("GetByChallengeID", mock.Anything, challengeID).Return([]*entity.Hint{}, nil)
}

func (h *BackupTestHelper) NewCompetition(name, mode string, allowTeamSwitch bool) *entity.Competition {
	h.t.Helper()
	return &entity.Competition{
		ID:              1,
		Name:            name,
		Mode:            entity.CompetitionMode(mode),
		AllowTeamSwitch: allowTeamSwitch,
	}
}

func (h *BackupTestHelper) NewChallenge(id uuid.UUID, title string, points int) *entity.Challenge {
	h.t.Helper()
	return &entity.Challenge{
		ID:         id,
		Title:      title,
		Points:     points,
		SolveCount: 0,
	}
}

func (h *BackupTestHelper) NewMinimalBackupData() *entity.BackupData {
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

func (h *BackupTestHelper) BuildBackupZip(data *entity.BackupData) ([]byte, int64) {
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
