package backup

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	importJobStartupRecoveryTimeout = 5 * time.Second
	importJobCleanupTimeout         = 30 * time.Second
	importStagingPrefix             = "imports/"
	importStagingContentType        = "application/zip"
)

func (uc *BackupUseCase) StartImportZIPJob(ctx context.Context, r io.Reader, size int64, opts domain.ImportOptions, archiveFilename string) (*domain.ImportJob, error) {
	if size < 0 {
		return nil, apperr.NewValidationErrorf("archive size must be non-negative")
	}

	if uc.deps.Storage == nil {
		return nil, apperr.NewValidationErrorf("backup storage is not configured")
	}

	jobID := uuid.New()
	stagingLocation := importStagingPrefix + jobID.String() + ".zip"

	filename := filepath.Base(archiveFilename)
	if filename == "" || filename == "." {
		filename = "backup.zip"
	}

	job := &domain.ImportJob{
		ID:              jobID,
		RequestedBy:     opts.AdminUserID,
		ClientIP:        opts.AdminIP,
		ArchiveFilename: filename,
		ArchiveSize:     size,
		StagingLocation: stagingLocation,
		Options:         opts,
		Status:          domain.ImportJobStatusQueued,
		Phase:           domain.ImportJobPhaseQueued,
	}

	created, err := uc.deps.BackupRepo.CreateImportJob(ctx, job)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - StartImportZIPJob - BackupRepo.CreateImportJob: %w", err)
	}

	if err := uc.deps.Storage.Upload(ctx, stagingLocation, r, size, importStagingContentType); err != nil {
		uc.failImportJob(ctx, created.ID, err.Error())
		uc.deleteStagedImportArchive(ctx, stagingLocation)

		return nil, fmt.Errorf("BackupUseCase - StartImportZIPJob - Storage.Upload: %w", err)
	}

	uc.jobs.Go(func() {
		uc.runImportJob(uc.importJobContext(ctx), created.ID)
	})

	return created, nil
}

func (uc *BackupUseCase) GetImportJob(ctx context.Context, id uuid.UUID) (*domain.ImportJob, error) {
	return uc.deps.BackupRepo.GetImportJob(ctx, id)
}

func (uc *BackupUseCase) runImportJob(ctx context.Context, jobID uuid.UUID) {
	defer func() {
		if recovered := recover(); recovered != nil {
			uc.failImportJob(ctx, jobID, fmt.Sprintf("import panic: %v", recovered))
		}
	}()

	job, err := uc.deps.BackupRepo.MarkImportJobRunning(ctx, jobID, domain.ImportJobPhaseValidating)
	if err != nil {
		if ctx.Err() != nil {
			uc.failImportJob(ctx, jobID, "import interrupted by backend shutdown")
		}

		uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"job_id": jobID}).Error("BackupUseCase - runImportJob - MarkImportJobRunning")

		return
	}

	tmp, size, err := uc.downloadStagedImportArchive(ctx, job.StagingLocation)
	if err != nil {
		uc.failImportJob(ctx, jobID, uc.importJobErrorMessage(ctx, err))
		uc.deleteStagedImportArchive(ctx, job.StagingLocation)

		return
	}

	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	result, err := uc.importZIP(ctx, tmp, size, job.Options, func(progressCtx context.Context, phase domain.ImportJobPhase) {
		if updateErr := uc.deps.BackupRepo.UpdateImportJobPhase(progressCtx, jobID, phase); updateErr != nil {
			uc.deps.Logger.WithError(updateErr).WithFields(logkit.Fields{"job_id": jobID, "phase": phase}).Warn("BackupUseCase - runImportJob - UpdateImportJobPhase")
		}
	})
	if err != nil {
		uc.failImportJob(ctx, jobID, uc.importJobErrorMessage(ctx, err))
		uc.deleteStagedImportArchive(ctx, job.StagingLocation)

		return
	}

	terminalCtx, terminalCancel := uc.importJobTerminalContext(ctx)
	defer terminalCancel()

	if err := uc.deps.BackupRepo.UpdateImportJobPhase(terminalCtx, jobID, domain.ImportJobPhaseCleanup); err != nil {
		uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"job_id": jobID}).Warn("BackupUseCase - runImportJob - cleanup phase")
	}

	uc.deleteStagedImportArchive(ctx, job.StagingLocation)

	if err := uc.deps.BackupRepo.CompleteImportJob(terminalCtx, jobID, result); err != nil {
		uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"job_id": jobID}).Error("BackupUseCase - runImportJob - CompleteImportJob")
	}
}

func (uc *BackupUseCase) importJobContext(ctx context.Context) context.Context {
	if uc.deps.StopContext != nil {
		return uc.deps.StopContext
	}

	return context.WithoutCancel(ctx)
}

func (uc *BackupUseCase) importJobTerminalContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), importJobCleanupTimeout)
}

func (uc *BackupUseCase) importJobErrorMessage(ctx context.Context, err error) string {
	if ctx.Err() != nil {
		return "import interrupted by backend shutdown: " + err.Error()
	}

	return err.Error()
}

func (uc *BackupUseCase) downloadStagedImportArchive(ctx context.Context, stagingLocation string) (*os.File, int64, error) {
	rc, err := uc.deps.Storage.Download(ctx, stagingLocation)
	if err != nil {
		return nil, 0, fmt.Errorf("BackupUseCase - downloadStagedImportArchive - Storage.Download: %w", err)
	}
	defer rc.Close()

	tmp, err := os.CreateTemp("", "astroctfb-import-*.zip")
	if err != nil {
		return nil, 0, fmt.Errorf("BackupUseCase - downloadStagedImportArchive - os.CreateTemp: %w", err)
	}

	size, err := io.Copy(tmp, rc)
	if err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())

		return nil, 0, fmt.Errorf("BackupUseCase - downloadStagedImportArchive - io.Copy: %w", err)
	}

	if _, err := tmp.Seek(0, 0); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())

		return nil, 0, fmt.Errorf("BackupUseCase - downloadStagedImportArchive - Seek: %w", err)
	}

	return tmp, size, nil
}

func (uc *BackupUseCase) failInterruptedImportJobs(ctx context.Context) {
	if ctx == nil || uc.deps.BackupRepo == nil {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), importJobStartupRecoveryTimeout)
	defer cancel()

	locations, err := uc.deps.BackupRepo.ListInterruptedImportJobStagingLocations(cleanupCtx)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("BackupUseCase - failInterruptedImportJobs - ListInterruptedImportJobStagingLocations")
	}

	if err := uc.deps.BackupRepo.FailInterruptedImportJobs(cleanupCtx); err != nil {
		uc.deps.Logger.WithError(err).Warn("BackupUseCase - failInterruptedImportJobs")

		return
	}

	for _, location := range uniqueNonEmptyStrings(locations) {
		uc.deleteStagedImportArchive(cleanupCtx, location)
	}
}

func (uc *BackupUseCase) failImportJob(ctx context.Context, jobID uuid.UUID, message string) {
	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), importJobCleanupTimeout)
	defer cancel()

	if err := uc.deps.BackupRepo.FailImportJob(cleanupCtx, jobID, message); err != nil {
		uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"job_id": jobID}).Error("BackupUseCase - failImportJob")
	}
}

func (uc *BackupUseCase) deleteStagedImportArchive(ctx context.Context, stagingLocation string) {
	if uc.deps.Storage == nil || stagingLocation == "" {
		return
	}

	cleanupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), importJobCleanupTimeout)
	defer cancel()

	if err := uc.deps.Storage.Delete(cleanupCtx, stagingLocation); err != nil {
		uc.deps.Logger.WithError(err).WithFields(logkit.Fields{"location": stagingLocation}).Warn("BackupUseCase - deleteStagedImportArchive")
	}
}

func uniqueNonEmptyStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))

	for _, value := range values {
		if value == "" {
			continue
		}

		if _, ok := seen[value]; ok {
			continue
		}

		seen[value] = struct{}{}
		result = append(result, value)
	}

	return result
}
