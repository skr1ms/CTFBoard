package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/skr1ms/CTFBoard/internal/repo"
)

type CleanupUseCase struct {
	teamRepo repo.TeamRepository
}

func NewCleanupUseCase(
	teamRepo repo.TeamRepository,
) *CleanupUseCase {
	return &CleanupUseCase{
		teamRepo: teamRepo,
	}
}

func (uc *CleanupUseCase) CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error {
	cutoffDate := time.Now().Add(-olderThan)
	if err := uc.teamRepo.HardDeleteTeams(ctx, cutoffDate); err != nil {
		return fmt.Errorf("CleanupUseCase - CleanupDeletedTeams: %w", err)
	}
	return nil
}
