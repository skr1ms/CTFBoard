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
// decodes the JSON into BackupData. The entry's uncompressed size is rejected
// before opening when it exceeds maxBackupJSONSize, and the decoder requires EOF
// after the first JSON document so crafted archives cannot hide trailing payloads.
// Returns ErrBackupJSONNotFound when no matching entry exists.
func (uc *BackupUseCase) importZIPReadBackup(zr *zip.Reader) (*domain.BackupData, error) {
	for _, f := range zr.File {
		if f.Name != "backup.json" {
			continue
		}

		if err := validateBackupJSONSize(f.UncompressedSize64); err != nil {
			return nil, err
		}

		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - open backup.json: %w", err)
		}
		defer rc.Close()

		limited := io.LimitReader(rc, maxBackupJSONSize+1)
		decoder := json.NewDecoder(limited)

		backupData := &domain.BackupData{}
		if err := decoder.Decode(backupData); err != nil {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - decode backup.json: %w", err)
		}

		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); err != io.EOF {
			return nil, fmt.Errorf("BackupUseCase - ImportZIP - backup.json contains trailing content")
		}

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

func validateBackupJSONSize(size uint64) error {
	if size > maxBackupJSONSize {
		return fmt.Errorf("BackupUseCase - ImportZIP - backup.json size %d exceeds limit %d", size, maxBackupJSONSize)
	}

	return nil
}

func validateUniqueZIPEntries(zr *zip.Reader) error {
	seen := make(map[string]struct{}, len(zr.File))

	for _, f := range zr.File {
		if _, ok := seen[f.Name]; ok {
			return fmt.Errorf("duplicate ZIP entry %q", f.Name)
		}

		seen[f.Name] = struct{}{}
	}

	return nil
}
