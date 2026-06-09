package backup

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/storagepath"
)

func backupFileZIPPath(file domain.File) (string, error) {
	filename, err := backupFileZIPName(file)
	if err != nil {
		return "", err
	}

	switch file.Type {
	case domain.FileTypeChallenge:
		if file.ChallengeID == nil {
			return "", fmt.Errorf("challenge_id is required")
		}

		return path.Join(backupFilesPrefix, "challenge-"+file.ChallengeID.String(), filename), nil
	case domain.FileTypeWriteup:
		if file.ChallengeID == nil {
			return "", fmt.Errorf("challenge_id is required")
		}

		return path.Join(backupFilesPrefix, "writeup-"+file.ChallengeID.String(), filename), nil
	case domain.FileTypePage:
		if file.PageID == nil {
			return "", fmt.Errorf("page_id is required")
		}

		return path.Join(backupFilesPrefix, "page-"+file.PageID.String(), filename), nil
	default:
		return "", fmt.Errorf("unsupported file type %q", file.Type)
	}
}

func backupFileZIPName(file domain.File) (string, error) {
	filename, _, err := backupFileSafeName(file)

	return filename, err
}

func backupFileSafeName(file domain.File) (string, bool, error) {
	if isSafeBackupFilename(file.Filename) {
		return file.Filename, false, nil
	}

	filename := storagepath.DownloadFilename(file.Location)
	if !isSafeBackupFilename(filename) {
		return "", false, fmt.Errorf("invalid filename")
	}

	return filename, true, nil
}

func normalizeBackupFileMetadata(file domain.File) (domain.File, bool, error) {
	filename, normalized, err := backupFileSafeName(file)
	if err != nil {
		return file, false, err
	}

	if file.Filename != filename {
		file.Filename = filename
		normalized = true
	}

	return file, normalized, nil
}

func isSafeBackupFilename(filename string) bool {
	if filename == "" ||
		filename == "." ||
		filename == string(filepath.Separator) ||
		strings.Contains(filename, "..") ||
		strings.ContainsAny(filename, `/\`) {
		return false
	}

	for _, r := range filename {
		if r < 0x20 || r == 0x7f {
			return false
		}
	}

	return true
}
