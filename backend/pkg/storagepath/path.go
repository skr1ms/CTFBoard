package storagepath

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

const (
	filePathHashLen       = 16
	legacyPathParts       = 2
	taskPathParts         = 3
	defaultDownloadName   = "download"
	taskPathPrefix        = "tasks/"
	taskPathPrefixNoSlash = "tasks"
)

var ErrInvalidFilename = errors.New("invalid storage filename")

var (
	validLegacyDownloadPathPattern = regexp.MustCompile(fmt.Sprintf(`^[a-f0-9]{%d}/.+$`, filePathHashLen))
	validTasksDownloadPathPattern  = regexp.MustCompile(fmt.Sprintf(`^%s/[a-f0-9]{%d}/.+$`, taskPathPrefixNoSlash, filePathHashLen))
)

func Generate(filename string) (string, error) {
	safeName := filepath.Base(filename)
	if safeName == "" || strings.Contains(safeName, "..") {
		return "", ErrInvalidFilename
	}

	var buf [filePathHashLen / 2]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("storagepath.Generate - crypto/rand: %w", err)
	}

	hash := hex.EncodeToString(buf[:])

	return path.Join("tasks", hash, safeName), nil
}

func ValidateDownloadPath(storagePath string) bool {
	if strings.Contains(storagePath, "..") {
		return false
	}

	return validLegacyDownloadPathPattern.MatchString(storagePath) || validTasksDownloadPathPattern.MatchString(storagePath)
}

func DownloadFilename(storagePath string) string {
	if strings.HasPrefix(storagePath, taskPathPrefix) {
		parts := strings.SplitN(storagePath, "/", taskPathParts)
		if len(parts) == taskPathParts {
			return filepath.Base(parts[2])
		}
	} else {
		parts := strings.SplitN(storagePath, "/", legacyPathParts)
		if len(parts) == legacyPathParts {
			return filepath.Base(parts[1])
		}
	}

	return defaultDownloadName
}
