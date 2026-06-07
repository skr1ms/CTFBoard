package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type importProgress func(ctx context.Context, phase domain.ImportJobPhase)

// ImportZIP imports a backup archive. It opens the ZIP, sums uncompressed sizes
// and rejects the archive when the ratio against the on-disk size or the
// absolute ceiling is exceeded (zip bomb protection). It then reads and decodes
// backup.json, validates the format version, and runs the database import inside
// a transaction (optionally erasing existing data first). After the transaction
// commits, challenge files are uploaded to object storage concurrently; storage
// uploads are intentionally outside the transaction because they are not
// transactional - failures are collected as warnings in ImportResult.Warnings
// rather than rolling back the database import.
func (uc *BackupUseCase) ImportZIP(ctx context.Context, r io.ReaderAt, size int64, opts domain.ImportOptions) (*domain.ImportResult, error) {
	return uc.importZIP(ctx, r, size, opts, nil)
}

func (uc *BackupUseCase) importZIP(ctx context.Context, r io.ReaderAt, size int64, opts domain.ImportOptions, progress importProgress) (*domain.ImportResult, error) {
	if progress != nil {
		progress(ctx, domain.ImportJobPhaseValidating)
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - NewReader: %w", err)
	}

	var totalUncompressed uint64

	for _, f := range zr.File {
		totalUncompressed += f.UncompressedSize64
	}

	if size < 0 {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: negative size")
	}

	maxAllowed := min(uint64(size)*maxUncompressedRatio, maxUncompressedAbsolute)

	if totalUncompressed > maxAllowed {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: uncompressed size %d exceeds limit %d (zip bomb protection)", totalUncompressed, maxAllowed)
	}

	backupData, err := uc.importZIPReadBackup(zr)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPReadBackup: %w", err)
	}

	if err := uc.importZIPValidateVersion(backupData); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPValidateVersion: %w", err)
	}

	preparedFiles, fileWarnings := uc.prepareImportFiles(zr, backupData.Files, opts.ValidateFiles)
	backupData.Files = preparedFiles

	result := &domain.ImportResult{
		Success:      true,
		Warnings:     fileWarnings,
		SkippedCount: len(fileWarnings),
	}

	if progress != nil {
		progress(ctx, domain.ImportJobPhaseImportingDB)
	}

	if err := uc.importZIPRunTx(ctx, backupData, opts); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPRunTx: %w", err)
	}
	// File upload to storage happens after the DB transaction commits intentionally
	// storage uploads are not transactional. If uploads fail, the DB records are kept
	// and the caller receives a partial result with SkippedCount > 0 so the issue is visible
	// A full rollback would require compensating deletes in DB, which adds complexity with
	// no meaningful safety gain since files can be re-uploaded manually
	if len(backupData.Files) > 0 {
		if progress != nil {
			progress(ctx, domain.ImportJobPhaseRestoringFiles)
		}

		fileErrors, err := uc.importFilesToStorage(ctx, zr, backupData.Files, opts)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("BackupUseCase - ImportZIP - importFilesToStorage")
		}

		if len(fileErrors) > 0 {
			result.Warnings = append(result.Warnings, fileErrors...)
			result.SkippedCount += len(fileErrors)
		}
	}

	uc.deps.Logger.Info("BackupUseCase - ImportZIP - completed", logkit.Fields{
		"challenges": len(backupData.Challenges),
		"teams":      len(backupData.Teams),
		"users":      len(backupData.Users),
		"files":      len(backupData.Files),
		"skipped":    result.SkippedCount,
	})

	return result, nil
}
