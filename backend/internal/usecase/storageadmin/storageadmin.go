package storageadmin

import (
	"context"
	"fmt"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type StoragePort interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, path string) error
}

type UseCase struct {
	storage StoragePort
}

type Deps struct {
	Storage StoragePort
}

var _ usecase.StorageAdminUseCase = (*UseCase)(nil)

func NewUseCase(deps Deps) *UseCase {
	return &UseCase{storage: deps.Storage}
}

func (uc *UseCase) List(ctx context.Context, prefix string) ([]string, error) {
	if err := validatePrefix(prefix); err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - validatePrefix: %w", err)
	}

	paths, err := uc.storage.List(ctx, prefix)
	if err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - Storage.List: %w", err)
	}

	return paths, nil
}

func (uc *UseCase) Delete(ctx context.Context, path string) error {
	if err := validatePath(path); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - validatePath: %w", err)
	}

	if err := uc.storage.Delete(ctx, path); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - Storage.Delete: %w", err)
	}

	return nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return nil
	}

	return validateStoragePath("prefix", prefix)
}

func validatePath(path string) error {
	if path == "" {
		return apperr.NewValidationErrorf("path is required")
	}

	return validateStoragePath("path", path)
}

func validateStoragePath(name, path string) error {
	if strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
		return apperr.NewValidationErrorf("invalid %s", name)
	}

	return nil
}
