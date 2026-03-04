package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
)

type CleanupUseCase struct {
	deps CleanupDeps
}

type CleanupDeps struct {
	TeamRepo repo.TeamRepository
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
