package backup

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
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

	if size < 0 {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: negative size")
	}

	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - NewReader: %w", err)
	}

	if err := validateZIPUncompressedSize(zr.File, size); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP: %w", err)
	}

	if err := validateUniqueZIPEntries(zr); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - validateUniqueZIPEntries: %w", err)
	}

	backupData, err := uc.importZIPReadBackup(zr)
	if err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPReadBackup: %w", err)
	}

	if err := uc.importZIPValidateVersion(backupData); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - importZIPValidateVersion: %w", err)
	}

	if err := validateBackupChallengeRequirements(backupData); err != nil {
		return nil, fmt.Errorf("BackupUseCase - ImportZIP - validateBackupChallengeRequirements: %w", err)
	}

	validateFiles := opts.ValidateFiles || opts.EraseExisting

	preparedFiles, fileWarnings := uc.prepareImportFiles(zr, backupData.Files, validateFiles)
	if opts.EraseExisting && len(fileWarnings) > 0 {
		return nil, apperr.NewValidationErrorf(
			"erase_existing import requires a complete valid file set: %s",
			summarizeImportWarnings(fileWarnings),
		)
	}

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

func summarizeImportWarnings(warnings []string) string {
	const maxShown = 3

	if len(warnings) <= maxShown {
		return strings.Join(warnings, "; ")
	}

	return fmt.Sprintf("%s; and %d more", strings.Join(warnings[:maxShown], "; "), len(warnings)-maxShown)
}

func validateZIPUncompressedSize(files []*zip.File, archiveSize int64) error {
	maxAllowed, err := maxZIPUncompressedAllowed(archiveSize)
	if err != nil {
		return err
	}

	var total uint64

	for _, f := range files {
		if f.UncompressedSize64 > maxAllowed || total > maxAllowed-f.UncompressedSize64 {
			return fmt.Errorf("uncompressed size exceeds limit %d (zip bomb protection)", maxAllowed)
		}

		total += f.UncompressedSize64
	}

	return nil
}

func maxZIPUncompressedAllowed(size int64) (uint64, error) {
	if size < 0 {
		return 0, fmt.Errorf("negative size")
	}

	archiveSize := uint64(size)
	absoluteLimit := uint64(maxUncompressedAbsolute)
	ratio := uint64(maxUncompressedRatio)

	if archiveSize > absoluteLimit/ratio {
		return absoluteLimit, nil
	}

	return min(archiveSize*ratio, absoluteLimit), nil
}

func validateBackupChallengeRequirements(data *domain.BackupData) error {
	if len(data.ChallengeRequirements) == 0 {
		return nil
	}

	challengeIDs := make(map[uuid.UUID]struct{}, len(data.Challenges))
	for _, ch := range data.Challenges {
		challengeIDs[ch.ID] = struct{}{}
	}

	adj := make(map[uuid.UUID][]uuid.UUID, len(data.ChallengeRequirements))

	for _, pair := range data.ChallengeRequirements {
		if _, ok := challengeIDs[pair.ChallengeID]; !ok {
			return fmt.Errorf("requirement references unknown challenge_id %s", pair.ChallengeID)
		}

		if _, ok := challengeIDs[pair.RequiredChallengeID]; !ok {
			return fmt.Errorf("requirement references unknown required_challenge_id %s", pair.RequiredChallengeID)
		}

		adj[pair.ChallengeID] = append(adj[pair.ChallengeID], pair.RequiredChallengeID)
	}

	if backupRequirementsContainCycle(adj) {
		return fmt.Errorf("requirements contain a cycle")
	}

	return nil
}

func backupRequirementsContainCycle(adj map[uuid.UUID][]uuid.UUID) bool {
	visiting := make(map[uuid.UUID]bool, len(adj))
	visited := make(map[uuid.UUID]bool, len(adj))

	var dfs func(uuid.UUID) bool

	dfs = func(node uuid.UUID) bool {
		if visiting[node] {
			return true
		}

		if visited[node] {
			return false
		}

		visiting[node] = true

		if slices.ContainsFunc(adj[node], dfs) {
			return true
		}

		visiting[node] = false
		visited[node] = true

		return false
	}

	for node := range adj {
		if dfs(node) {
			return true
		}
	}

	return false
}
