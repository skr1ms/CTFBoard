package backup

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	backupMock "github.com/TakuyaYagam1/AstroCTFb/internal/usecase/backup/mock"
)

func TestBackupUseCase_StartImportZIPJob_ActiveConflictSkipsUpload(t *testing.T) {
	t.Parallel()

	repo := backupMock.NewMockBackupRepository(t)
	storage := backupMock.NewMockBackupStorage(t)
	uc := NewBackupUseCase(BackupDeps{BackupRepo: repo, Storage: storage})

	repo.EXPECT().
		CreateImportJob(mock.Anything, mock.Anything).
		Return(nil, apperr.ErrBackupImportAlreadyRunning).
		Once()

	job, err := uc.StartImportZIPJob(context.Background(), strings.NewReader("PKzip"), 5, domain.ImportOptions{}, "backup.zip")

	require.Error(t, err)
	assert.Nil(t, job)
	assert.ErrorIs(t, err, apperr.ErrBackupImportAlreadyRunning)
	storage.AssertNotCalled(t, "Upload", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
}

func TestBackupUseCase_StartImportZIPJob_UploadFailureFailsJobAndDeletesStaging(t *testing.T) {
	t.Parallel()

	repo := backupMock.NewMockBackupRepository(t)
	storage := backupMock.NewMockBackupStorage(t)
	uc := NewBackupUseCase(BackupDeps{BackupRepo: repo, Storage: storage})

	var (
		jobID           uuid.UUID
		stagingLocation string
	)

	repo.EXPECT().
		CreateImportJob(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, job *domain.ImportJob) (*domain.ImportJob, error) {
			jobID = job.ID
			stagingLocation = job.StagingLocation

			return job, nil
		}).
		Once()
	storage.EXPECT().
		Upload(mock.Anything, mock.MatchedBy(func(path string) bool { return path == stagingLocation }), mock.Anything, int64(5), importStagingContentType).
		Return(errors.New("storage unavailable")).
		Once()
	repo.EXPECT().
		FailImportJob(mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool { return id == jobID }), "storage unavailable").
		Return(nil).
		Once()
	storage.EXPECT().
		Delete(mock.Anything, mock.MatchedBy(func(path string) bool { return path == stagingLocation })).
		Return(nil).
		Once()

	job, err := uc.StartImportZIPJob(context.Background(), strings.NewReader("PKzip"), 5, domain.ImportOptions{}, "backup.zip")

	require.Error(t, err)
	assert.Nil(t, job)
	assert.Contains(t, err.Error(), "Storage.Upload")
}

func TestBackupUseCase_StartImportZIPJob_CanceledStopContextFailsQueuedJob(t *testing.T) {
	t.Parallel()

	stopCtx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := backupMock.NewMockBackupRepository(t)
	storage := backupMock.NewMockBackupStorage(t)

	var jobID uuid.UUID

	repo.EXPECT().
		ListInterruptedImportJobStagingLocations(mock.Anything).
		Return([]string{"imports/interrupted.zip"}, nil).
		Once()
	repo.EXPECT().
		FailInterruptedImportJobs(mock.Anything).
		Return(nil).
		Once()
	storage.EXPECT().
		Delete(mock.Anything, "imports/interrupted.zip").
		Return(nil).
		Once()
	uc := NewBackupUseCase(BackupDeps{StopContext: stopCtx, BackupRepo: repo, Storage: storage})
	repo.EXPECT().
		CreateImportJob(mock.Anything, mock.Anything).
		RunAndReturn(func(_ context.Context, job *domain.ImportJob) (*domain.ImportJob, error) {
			jobID = job.ID

			return job, nil
		}).
		Once()
	storage.EXPECT().
		Upload(mock.Anything, mock.Anything, mock.Anything, int64(5), importStagingContentType).
		Return(nil).
		Once()
	repo.EXPECT().
		MarkImportJobRunning(mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool { return id == jobID }), domain.ImportJobPhaseValidating).
		Return(nil, context.Canceled).
		Once()
	repo.EXPECT().
		FailImportJob(mock.Anything, mock.MatchedBy(func(id uuid.UUID) bool { return id == jobID }), "import interrupted by backend shutdown").
		Return(nil).
		Once()

	job, err := uc.StartImportZIPJob(context.Background(), strings.NewReader("PKzip"), 5, domain.ImportOptions{}, "backup.zip")
	require.NoError(t, err)
	require.NotNil(t, job)

	uc.Wait()
}

func TestBackupUseCase_ImportJobTerminalContextIgnoresParentCancellation(t *testing.T) {
	t.Parallel()

	parent, cancelParent := context.WithCancel(context.Background())
	cancelParent()

	uc := &BackupUseCase{}

	ctx, cancel := uc.importJobTerminalContext(parent)
	defer cancel()

	require.NoError(t, ctx.Err())
	deadline, ok := ctx.Deadline()
	require.True(t, ok)
	require.WithinDuration(t, time.Now().Add(importJobCleanupTimeout), deadline, time.Second)
}
