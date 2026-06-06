package backup

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	maxUncompressedRatio    = 10
	maxUncompressedAbsolute = 2 * 1024 * 1024 * 1024
	maxBackupJSONSize       = 100 * 1024 * 1024
)

// importZIPReadBackup scans the ZIP entries for "backup.json", opens it, and
// decodes the JSON into BackupData. Reading is limited to maxBackupJSONSize to
// prevent excessive memory use from a malformed archive. Returns
// ErrBackupJSONNotFound when no matching entry exists.
func (uc *BackupUseCase) importZIPReadBackup(zr *zip.Reader) (*domain.BackupData, error) {
	for _, f := range zr.File {
		if f.Name != "backup.json" {
			continue
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - open backup.json: %w", err)
		}

		limited := io.LimitReader(rc, maxBackupJSONSize)

		backupData := &domain.BackupData{}
		if err := json.NewDecoder(limited).Decode(backupData); err != nil {
			_ = rc.Close()

			return nil, fmt.Errorf("BackupUseCase - ImportZIP - decode backup.json: %w", err)
		}

		_ = rc.Close()

		return backupData, nil
	}

	return nil, apperr.ErrBackupJSONNotFound
}

// importZIPValidateVersion checks that the backup's Version field matches
// domain.BackupVersion. Mismatches return ErrBackupVersionUnsupported so the
// caller can surface a clear error rather than attempting a broken import.
func (uc *BackupUseCase) importZIPValidateVersion(backupData *domain.BackupData) error {
	if backupData.Version != domain.BackupVersion {
		return apperr.ErrBackupVersionUnsupported
	}

	return nil
}
