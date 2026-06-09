package request

import (
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

// ValidateStoragePath allows relative object keys only and rejects traversal.
func ValidateStoragePath(path string) error {
	if path == "." {
		return apperr.NewValidationErrorf("invalid path")
	}

	if strings.Contains(path, "..") {
		return apperr.NewValidationErrorf("invalid path")
	}

	if strings.Contains(path, "\\") {
		return apperr.NewValidationErrorf("invalid path")
	}

	if strings.HasPrefix(path, "/") {
		return apperr.NewValidationErrorf("invalid path")
	}

	return nil
}

func ValidateStoragePrefix(prefix string) error {
	if prefix == "" {
		return apperr.NewValidationErrorf("prefix is required")
	}

	if err := ValidateStoragePath(prefix); err != nil {
		return apperr.NewValidationErrorf("invalid prefix")
	}

	return nil
}

func StorageAdminListParams(params openapi.GetAdminStorageParams) (usecase.StorageAdminListParams, error) {
	if err := ValidateStoragePrefix(params.Prefix); err != nil {
		return usecase.StorageAdminListParams{}, err
	}

	limit := 0

	if params.Limit != nil {
		limit = *params.Limit
	}

	return usecase.StorageAdminListParams{
		Prefix: params.Prefix,
		Limit:  limit,
	}, nil
}

func StorageAdminDeleteParams(path string, actorID uuid.UUID, clientIP string) (usecase.StorageAdminDeleteParams, error) {
	if err := ValidateStoragePath(path); err != nil {
		return usecase.StorageAdminDeleteParams{}, err
	}

	return usecase.StorageAdminDeleteParams{
		Path:     path,
		ActorID:  actorID,
		ClientIP: clientIP,
	}, nil
}
