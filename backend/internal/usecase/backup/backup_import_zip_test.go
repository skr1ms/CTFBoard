package backup

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	logMock "github.com/wahrwelt-kit/go-logkit/mock"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
)

func TestBackupUseCase_ImportZIP_Success(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)
	backupRepo := backupMock.NewMockBackupRepository(t)
	log := logMock.NewMockLogger(t)

	backupData := &domain.BackupData{
		Version:     domain.BackupVersion,
		ExportedAt:  time.Now().UTC(),
		Competition: &domain.Competition{Name: "Test", Mode: "teams_only"},
	}
	zipBuf := new(bytes.Buffer)
	zw := zip.NewWriter(zipBuf)
	w, err := zw.Create("backup.json")
	require.NoError(t, err)
	require.NoError(t, json.NewEncoder(w).Encode(backupData))
	require.NoError(t, zw.Close())

	tm.EXPECT().Run(mock.Anything, mock.Anything).RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
		return fn(ctx)
	}).Once()
	backupRepo.EXPECT().ImportCompetition(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportTags(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportTopics(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallenges(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallengeTags(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallengeTopics(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportBrackets(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportUsers(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportTeams(mock.Anything, mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().UpdateUserTeamIDs(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportAwards(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportSolves(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportHintUnlocks(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFileMetadata(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportChallengeRequirements(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportSolutions(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportRatings(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportComments(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFields(mock.Anything, mock.Anything).Return(nil).Once()
	backupRepo.EXPECT().ImportFieldValues(mock.Anything, mock.Anything).Return(nil).Once()
	log.EXPECT().Info(mock.Anything, mock.Anything).Maybe()

	deps := BackupDeps{
		TM:         tm,
		BackupRepo: backupRepo,
		Logger:     log,
	}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	r := bytes.NewReader(zipBuf.Bytes())
	readerAt := io.NewSectionReader(r, 0, int64(zipBuf.Len()))

	result, err := uc.ImportZIP(ctx, readerAt, int64(zipBuf.Len()), domain.ImportOptions{})

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestBackupUseCase_ImportZIP_Error_InvalidZIP(t *testing.T) {
	t.Parallel()
	log := logMock.NewMockLogger(t)

	deps := BackupDeps{Logger: log}
	uc := NewBackupUseCase(deps)

	ctx := context.Background()
	invalidZIP := bytes.NewReader([]byte("not a zip file"))
	readerAt := io.NewSectionReader(invalidZIP, 0, 13)

	_, err := uc.ImportZIP(ctx, readerAt, 13, domain.ImportOptions{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "BackupUseCase - ImportZIP")
}

func TestBackupUseCase_ImportZIP_Error_MissingBackupJSON(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)

	zipBuf := newBackupZip(t, map[string][]byte{
		"README.md": []byte("backup metadata is missing"),
	})

	uc := NewBackupUseCase(BackupDeps{TM: tm})
	readerAt := io.NewSectionReader(bytes.NewReader(zipBuf.Bytes()), 0, int64(zipBuf.Len()))

	_, err := uc.ImportZIP(context.Background(), readerAt, int64(zipBuf.Len()), domain.ImportOptions{})

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrBackupJSONNotFound))
}

func TestBackupUseCase_ImportZIP_Error_UnsupportedVersion(t *testing.T) {
	t.Parallel()
	tm := backupMock.NewMockTransactionManager(t)

	backupData := &domain.BackupData{
		Version:     "0.9",
		ExportedAt:  time.Now().UTC(),
		Competition: &domain.Competition{Name: "Old", Mode: "teams_only"},
	}
	payload, err := json.Marshal(backupData)
	require.NoError(t, err)

	zipBuf := newBackupZip(t, map[string][]byte{
		"backup.json": payload,
	})

	uc := NewBackupUseCase(BackupDeps{TM: tm})
	readerAt := io.NewSectionReader(bytes.NewReader(zipBuf.Bytes()), 0, int64(zipBuf.Len()))

	_, err = uc.ImportZIP(context.Background(), readerAt, int64(zipBuf.Len()), domain.ImportOptions{})

	require.Error(t, err)
	assert.True(t, errors.Is(err, apperr.ErrBackupVersionUnsupported))
}

func newBackupZip(t *testing.T, entries map[string][]byte) *bytes.Buffer {
	t.Helper()

	zipBuf := new(bytes.Buffer)
	zw := zip.NewWriter(zipBuf)

	for name, content := range entries {
		w, err := zw.Create(name)
		require.NoError(t, err)
		_, err = w.Write(content)
		require.NoError(t, err)
	}

	require.NoError(t, zw.Close())

	return zipBuf
}
