package competition

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (h *CompetitionTestHelper) CreateBackupUseCase() *BackupUseCase {
	h.t.Helper()
	return NewBackupUseCase(BackupDeps{
		CompetitionRepo: h.deps.competitionRepo,
		ChallengeRepo:   h.deps.challengeRepo,
		HintRepo:        h.deps.hintRepo,
		TeamRepo:        h.deps.teamRepo,
		UserRepo:        h.deps.userRepo,
		AwardRepo:       h.deps.awardRepo,
		SolveRepo:       h.deps.solveRepo,
		FileRepo:        nil,
		BackupRepo:      h.deps.backupRepo,
		Storage:         nil,
		TxRepo:          h.deps.txRepo,
		Logger:          h.deps.logger,
	})
}

func (h *CompetitionTestHelper) SetupBackupExportMocks(comp *entity.Competition, challenges []*repo.ChallengeWithSolved, challengeID uuid.UUID) {
	h.t.Helper()
	h.deps.competitionRepo.On("Get", mock.Anything).Return(comp, nil)
	h.deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return(challenges, nil)
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
