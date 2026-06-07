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
	filename := filepath.Base(file.Filename)
	if !isSafeBackupFilename(filename) {
		filename = storagepath.DownloadFilename(file.Location)
	}

	if !isSafeBackupFilename(filename) {
		return "", fmt.Errorf("invalid filename")
	}

	return filename, nil
}

func isSafeBackupFilename(filename string) bool {
	return filename != "" &&
		filename != "." &&
		filename != string(filepath.Separator) &&
		!strings.Contains(filename, "..")
}
