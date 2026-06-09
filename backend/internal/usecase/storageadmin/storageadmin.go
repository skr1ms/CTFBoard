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
	List(ctx context.Context, prefix string, limit int) ([]string, error)
	ListPage(ctx context.Context, prefix, cursor string, limit int) ([]string, string, error)
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

const (
	defaultStorageListLimit = 500
	maxStorageListLimit     = 1000
)

func NewUseCase(deps Deps) *UseCase {
	return &UseCase{storage: deps.Storage, auditLog: deps.AuditLog}
}

func (uc *UseCase) List(ctx context.Context, params usecase.StorageAdminListParams) (*usecase.StorageAdminListResult, error) {
	if err := validatePrefix(params.Prefix); err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - validatePrefix: %w", err)
	}

	if err := validateCursor(params.Cursor); err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - validateCursor: %w", err)
	}

	limit, err := normalizeListLimit(params.Limit)
	if err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - normalizeListLimit: %w", err)
	}

	paths, nextCursor, err := uc.storage.ListPage(ctx, params.Prefix, params.Cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("StorageAdminUseCase - List - Storage.ListPage: %w", err)
	}

	return &usecase.StorageAdminListResult{Paths: paths, NextCursor: nextCursor}, nil
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

	auditLog := &domain.AuditLog{
		UserID:     &params.ActorID,
		Action:     domain.AuditActionDelete,
		EntityType: domain.AuditEntityStorage,
		EntityID:   storageAuditEntityID,
		IP:         params.ClientIP,
		Details: map[string]any{
			"message": "storage object delete requested",
			"path":    params.Path,
			"status":  "requested",
		},
	}
	if err := uc.auditLog.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - AuditLogRepo.Create: %w", err)
	}

	if err := uc.storage.Delete(ctx, params.Path); err != nil {
		return fmt.Errorf("StorageAdminUseCase - Delete - Storage.Delete: %w", err)
	}

	return nil
}

func validatePrefix(prefix string) error {
	if prefix == "" {
		return apperr.NewValidationErrorf("prefix is required")
	}

	return validateStoragePath("prefix", prefix)
}

func normalizeListLimit(limit int) (int, error) {
	if limit == 0 {
		return defaultStorageListLimit, nil
	}

	if limit < 0 || limit > maxStorageListLimit {
		return 0, apperr.NewValidationErrorf("limit must be between 1 and %d", maxStorageListLimit)
	}

	return limit, nil
}

func validatePath(path string) error {
	if path == "" {
		return apperr.NewValidationErrorf("path is required")
	}

	return validateStoragePath("path", path)
}

func validateCursor(cursor string) error {
	if cursor == "" {
		return nil
	}

	return validateStoragePath("cursor", cursor)
}

func validateStoragePath(name, path string) error {
	if path == "." || strings.Contains(path, "..") || strings.Contains(path, "\\") || strings.HasPrefix(path, "/") {
		return apperr.NewValidationErrorf("invalid %s", name)
	}

	return nil
}
