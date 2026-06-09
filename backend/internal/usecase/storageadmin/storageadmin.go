package storageadmin

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

type StoragePort interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, path string) error
}

type UseCase struct {
	storage  StoragePort
	auditLog repo.AuditLogRepository
}

type Deps struct {
	Storage  StoragePort
	AuditLog repo.AuditLogRepository
}

var _ usecase.StorageAdminUseCase = (*UseCase)(nil)

const storageAuditEntityID = "object"

func NewUseCase(deps Deps) *UseCase {
	return &UseCase{storage: deps.Storage, auditLog: deps.AuditLog}
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

func (uc *UseCase) Delete(ctx context.Context, params usecase.StorageAdminDeleteParams) error {
	if err := validatePath(params.Path); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - validatePath: %w", err)
	}

	if params.ActorID == uuid.Nil {
		return apperr.NewValidationErrorf("actor_id is required")
	}

	if uc.auditLog == nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - AuditLogRepo not configured")
	}

	if err := uc.storage.Delete(ctx, params.Path); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - Storage.Delete: %w", err)
	}

	auditLog := &domain.AuditLog{
		UserID:     &params.ActorID,
		Action:     domain.AuditActionDelete,
		EntityType: domain.AuditEntityStorage,
		EntityID:   storageAuditEntityID,
		IP:         params.ClientIP,
		Details: map[string]any{
			"message": "storage object deleted",
			"path":    params.Path,
		},
	}
	if err := uc.auditLog.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - AuditLogRepo.Create: %w", err)
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
