package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	"github.com/TakuyaYagam1/AstroCTFb/internal/storage"
)

type CleanupUseCase struct {
	deps CleanupDeps
}

type CleanupDeps struct {
	TeamRepo repo.TeamRepository
	FileRepo repo.FileRepository
	Storage  storage.Provider
}

var _ Cleaner = (*CleanupUseCase)(nil)

func NewCleanupUseCase(deps CleanupDeps) *CleanupUseCase {
	return &CleanupUseCase{deps: deps}
}

func (uc *CleanupUseCase) CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error {
	cutoffDate := time.Now().Add(-olderThan)
	if err := uc.deps.TeamRepo.HardDeleteTeams(ctx, cutoffDate); err != nil {
		return fmt.Errorf("CleanupUseCase - CleanupDeletedTeams - TeamRepo.HardDeleteTeams: %w", err)
	}
	return nil
}

const cleanupLocationsBatchSize = 1000

func (uc *CleanupUseCase) CleanupOrphanedStorageFiles(ctx context.Context, prefix string) (deleted int, err error) {
	if uc.deps.FileRepo == nil || uc.deps.Storage == nil {
		return 0, nil
	}
	known := make(map[string]struct{})
	for offset := 0; ; offset += cleanupLocationsBatchSize {
		locs, listErr := uc.deps.FileRepo.ListLocations(ctx, cleanupLocationsBatchSize, offset)
		if listErr != nil {
			return deleted, fmt.Errorf("CleanupUseCase - CleanupOrphanedStorageFiles - FileRepo.ListLocations: %w", listErr)
		}
		if len(locs) == 0 {
			break
		}
		for _, loc := range locs {
			known[loc] = struct{}{}
		}
	}
	paths, err := uc.deps.Storage.List(ctx, prefix)
	if err != nil {
		return 0, fmt.Errorf("CleanupUseCase - CleanupOrphanedStorageFiles - Storage.List: %w", err)
	}
	for _, path := range paths {
		if _, ok := known[path]; ok {
			continue
		}
		if delErr := uc.deps.Storage.Delete(ctx, path); delErr != nil {
			err = errors.Join(err, fmt.Errorf("CleanupUseCase - CleanupOrphanedStorageFiles - Storage.Delete %q: %w", path, delErr))
			continue
		}
		deleted++
	}
	return deleted, err
}
