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
	taskPathParts         = 3
	defaultDownloadName   = "download"
	taskPathPrefixNoSlash = "tasks"
)

var ErrInvalidFilename = errors.New("invalid storage filename")

var validTasksDownloadPathPattern = regexp.MustCompile(fmt.Sprintf(`^%s/[a-f0-9]{%d}/[^/]+$`, taskPathPrefixNoSlash, filePathHashLen))

func Generate(filename string) (string, error) {
	safeName := filepath.Base(filename)
	if safeName == "" || safeName == "." || safeName != filename ||
		strings.Contains(safeName, "..") || strings.ContainsAny(safeName, `/\`) {
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

	return validTasksDownloadPathPattern.MatchString(storagePath)
}

func DownloadFilename(storagePath string) string {
	if !ValidateDownloadPath(storagePath) {
		return defaultDownloadName
	}

	parts := strings.Split(storagePath, "/")
	if len(parts) == taskPathParts {
		return filepath.Base(parts[2])
	}

	return defaultDownloadName
}
