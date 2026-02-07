package competition

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestBackupUseCase_Export_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc := h.CreateBackupUseCase()

	comp := h.NewCompetition("CTF", "flexible", true)
	challengeID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		{Challenge: &entity.Challenge{ID: challengeID, Title: "Chall", Description: "Desc", Category: "Web", Points: 100}, Solved: false},
	}

	h.SetupBackupExportMocks(comp, challenges, challengeID)

	opts := entity.ExportOptions{IncludeTeams: false, IncludeUsers: false, IncludeAwards: false, IncludeSolves: false}
	result, err := uc.Export(context.Background(), opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, entity.BackupVersion, result.Version)
	assert.NotZero(t, result.ExportedAt)
	assert.Equal(t, comp.Name, result.Competition.Name)
	assert.Len(t, result.Challenges, 1)
	assert.Equal(t, "Chall", result.Challenges[0].Title)
}

func TestBackupUseCase_Export_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc := h.CreateBackupUseCase()

	expectedErr := errors.New("db error")
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, expectedErr)
	deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return([]*repo.ChallengeWithSolved{}, nil)

	result, err := uc.Export(context.Background(), entity.ExportOptions{})

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "BackupUseCase")
	assert.Contains(t, err.Error(), expectedErr.Error())
}

func TestBackupUseCase_ExportZIP_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	uc := h.CreateBackupUseCase()

	comp := h.NewCompetition("CTF", "flexible", true)
	challengeID := uuid.New()
	challenges := []*repo.ChallengeWithSolved{
		{Challenge: &entity.Challenge{ID: challengeID, Title: "Chall", Points: 100}, Solved: false},
	}
	h.SetupBackupExportMocks(comp, challenges, challengeID)

	opts := entity.ExportOptions{IncludeTeams: false, IncludeUsers: false, IncludeAwards: false, IncludeSolves: false, IncludeFiles: false}
	rc, err := uc.ExportZIP(context.Background(), opts)

	assert.NoError(t, err)
	assert.NotNil(t, rc)
	defer rc.Close()

	data, err := io.ReadAll(rc)
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	assert.NoError(t, err)

	var foundBackup bool
	for _, f := range zr.File {
		if f.Name == "backup.json" {
			foundBackup = true
			rc, err := f.Open()
			require.NoError(t, err)
			var backup entity.BackupData
			assert.NoError(t, json.NewDecoder(rc).Decode(&backup))
			rc.Close()
			assert.Equal(t, entity.BackupVersion, backup.Version)
			assert.Len(t, backup.Challenges, 1)
			break
		}
	}
	assert.True(t, foundBackup)
}

func TestBackupUseCase_ExportZIP_Error(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc := h.CreateBackupUseCase()

	expectedErr := errors.New("db error")
	deps.competitionRepo.On("Get", mock.Anything).Return(nil, expectedErr)
	deps.challengeRepo.On("GetAll", mock.Anything, (*uuid.UUID)(nil), (*uuid.UUID)(nil)).Return([]*repo.ChallengeWithSolved{}, nil)

	rc, err := uc.ExportZIP(context.Background(), entity.ExportOptions{})
	assert.NoError(t, err)
	assert.NotNil(t, rc)
	defer rc.Close()

	_, err = io.ReadAll(rc)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase")
}

func TestBackupUseCase_ImportZIP_Success(t *testing.T) {
	h := NewCompetitionTestHelper(t)
	deps := h.Deps()
	uc := h.CreateBackupUseCase()

	data := h.NewMinimalBackupData()
	zipBytes, zipSize := h.BuildBackupZip(data)

	deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		ctx := args.Get(0).(context.Context)                              //nolint:errcheck
		fn := args.Get(1).(func(context.Context, repo.Transaction) error) //nolint:errcheck
		_ = fn(ctx, nil)                                                  //nolint:errcheck
	})
	deps.backupRepo.On("ImportCompetitionTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportChallengesTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportTeamsTx", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportUsersTx", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportAwardsTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportSolvesTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	deps.backupRepo.On("ImportFileMetadataTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	opts := entity.ImportOptions{EraseExisting: false, ConflictMode: entity.ConflictModeOverwrite}
	result, err := uc.ImportZIP(context.Background(), bytes.NewReader(zipBytes), zipSize, opts)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestBackupUseCase_ImportZIP_Error(t *testing.T) {
	t.Run("invalid_zip", func(t *testing.T) {
		h := NewCompetitionTestHelper(t)
		uc := h.CreateBackupUseCase()

		invalidZip := []byte("not a zip")
		result, err := uc.ImportZIP(context.Background(), bytes.NewReader(invalidZip), int64(len(invalidZip)), entity.ImportOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "BackupUseCase")
	})

	t.Run("backup_json_not_found", func(t *testing.T) {
		h := NewCompetitionTestHelper(t)
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		_, err := zw.Create("other.txt")
		require.NoError(t, err)
		_ = zw.Close()
		b := buf.Bytes()

		uc := h.CreateBackupUseCase()
		result, err := uc.ImportZIP(context.Background(), bytes.NewReader(b), int64(len(b)), entity.ImportOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "backup.json not found")
	})

	t.Run("tx_error", func(t *testing.T) {
		h := NewCompetitionTestHelper(t)
		deps := h.Deps()
		uc := h.CreateBackupUseCase()

		data := h.NewMinimalBackupData()
		zipBytes, zipSize := h.BuildBackupZip(data)

		expectedErr := errors.New("tx failed")
		deps.txRepo.On("RunTransaction", mock.Anything, mock.Anything).Return(expectedErr)

		result, err := uc.ImportZIP(context.Background(), bytes.NewReader(zipBytes), zipSize, entity.ImportOptions{})

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), expectedErr.Error())
	})
}
