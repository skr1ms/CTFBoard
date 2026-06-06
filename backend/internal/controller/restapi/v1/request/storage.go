package request

import (
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

// ValidateStoragePath allows relative object keys only and rejects traversal.
func ValidateStoragePath(path string) error {
	if strings.Contains(path, "..") {
		return apperr.NewValidationErrorf("invalid path")
	}

	if strings.HasPrefix(path, "/") {
		return apperr.NewValidationErrorf("invalid path")
	}

	return nil
}

func ValidateStoragePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}

	if err := ValidateStoragePath(prefix); err != nil {
		return apperr.NewValidationErrorf("invalid prefix")
	}

	return nil
}
