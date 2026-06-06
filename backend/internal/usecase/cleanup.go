package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

type CleanupUseCase struct {
	deps CleanupDeps
}

type CleanupStorage interface {
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, path string) error
}

type CleanupDeps struct {
	UserRepo     repo.UserRepository
	TeamRepo     repo.TeamRepository
	FileRepo     repo.FileRepository
	Storage      CleanupStorage
	TrackingRepo repo.TrackingRepository
}

var _ Cleaner = (*CleanupUseCase)(nil)

func NewCleanupUseCase(deps CleanupDeps) *CleanupUseCase {
	return &CleanupUseCase{deps: deps}
}

// CleanupDeletedTeams hard-deletes soft-deleted team records older than olderThan.
// Soft-deleted teams are those with a non-null deleted_at set before the cutoff.
func (uc *CleanupUseCase) CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error {
	cutoffDate := time.Now().Add(-olderThan)

	err := uc.deps.TeamRepo.HardDeleteTeams(ctx, cutoffDate)
	if err != nil {
		return fmt.Errorf("CleanupUseCase - CleanupDeletedTeams - TeamRepo.HardDeleteTeams: %w", err)
	}

	return nil
}

const (
	cleanupLocationsBatchSize = 1000
	cleanupUsersPrefix        = "users/"
	cleanupTeamsPrefix        = "teams/"
)

// CleanupOrphanedStorageFiles removes objects from storage that have no
// corresponding database record. It pages through all known file locations in
// batches of cleanupLocationsBatchSize, building an in-memory set, then lists
// every object under prefix from storage and deletes any path not present in
// the set. Deletion errors are joined and returned alongside the count of
// successfully deleted objects so the caller can log partial failures without
// losing the total progress.
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

		delErr := uc.deps.Storage.Delete(ctx, path)
		if delErr != nil {
			err = errors.Join(err, fmt.Errorf("CleanupUseCase - CleanupOrphanedStorageFiles - Storage.Delete %q: %w", path, delErr))

			continue
		}

		deleted++
	}

	return deleted, err
}

// CleanupOrphanedAvatars removes avatar objects from storage that are not
// referenced by any user or team record. It builds a set of valid paths from
// all non-empty avatar URLs returned by the user and team repositories, adding
// both the full-resolution path and the thumbnail path (derived via
// thumbPathFromFull) for each URL. It then lists all objects under the "users/"
// and "teams/" storage prefixes and deletes any object whose path is not in the
// valid set. Deletion errors are joined and returned alongside the deleted count.
func (uc *CleanupUseCase) CleanupOrphanedAvatars(ctx context.Context) (int, error) {
	if uc.deps.UserRepo == nil || uc.deps.TeamRepo == nil || uc.deps.Storage == nil {
		return 0, nil
	}

	userAvatars, err := uc.deps.UserRepo.ListAllUserAvatarURLs(ctx)
	if err != nil {
		return 0, fmt.Errorf("CleanupUseCase - CleanupOrphanedAvatars - UserRepo.ListAllUserAvatarURLs: %w", err)
	}

	teamAvatars, err := uc.deps.TeamRepo.ListAllTeamAvatarURLs(ctx)
	if err != nil {
		return 0, fmt.Errorf("CleanupUseCase - CleanupOrphanedAvatars - TeamRepo.ListAllTeamAvatarURLs: %w", err)
	}

	validPaths := make(map[string]struct{})

	for _, url := range userAvatars {
		if url != nil && *url != "" {
			validPaths[*url] = struct{}{}
			validPaths[domain.ThumbPathFromFull(*url)] = struct{}{}
		}
	}

	for _, url := range teamAvatars {
		if url != nil && *url != "" {
			validPaths[*url] = struct{}{}
			validPaths[domain.ThumbPathFromFull(*url)] = struct{}{}
		}
	}

	userFiles, err := uc.deps.Storage.List(ctx, cleanupUsersPrefix)
	if err != nil {
		return 0, fmt.Errorf("CleanupUseCase - CleanupOrphanedAvatars - Storage.List(users/): %w", err)
	}

	teamFiles, err := uc.deps.Storage.List(ctx, cleanupTeamsPrefix)
	if err != nil {
		return 0, fmt.Errorf("CleanupUseCase - CleanupOrphanedAvatars - Storage.List(teams/): %w", err)
	}

	allFiles := make([]string, 0, len(userFiles)+len(teamFiles))
	allFiles = append(allFiles, userFiles...)
	allFiles = append(allFiles, teamFiles...)
	deleted := 0

	var errs error

	for _, filePath := range allFiles {
		if _, exists := validPaths[filePath]; exists {
			continue
		}

		delErr := uc.deps.Storage.Delete(ctx, filePath)
		if delErr != nil {
			errs = errors.Join(errs, fmt.Errorf("CleanupUseCase - CleanupOrphanedAvatars - Storage.Delete %q: %w", filePath, delErr))

			continue
		}

		deleted++
	}

	return deleted, errs
}

// CleanupOldTracking deletes page-view tracking records and challenge-open events
// older than olderThan. Both tables are pruned in separate calls; no-ops when
// TrackingRepo is not wired.
func (uc *CleanupUseCase) CleanupOldTracking(ctx context.Context, olderThan time.Duration) error {
	if uc.deps.TrackingRepo == nil {
		return nil
	}

	cutoffDate := time.Now().Add(-olderThan)

	if err := uc.deps.TrackingRepo.DeleteOlderThan(ctx, cutoffDate); err != nil {
		return fmt.Errorf("CleanupUseCase - CleanupOldTracking - DeleteOlderThan: %w", err)
	}

	if err := uc.deps.TrackingRepo.DeleteChallengeOpensOlderThan(ctx, cutoffDate); err != nil {
		return fmt.Errorf("CleanupUseCase - CleanupOldTracking - DeleteChallengeOpensOlderThan: %w", err)
	}

	return nil
}
