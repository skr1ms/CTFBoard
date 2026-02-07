package usecase

import (
	"context"
	"time"

	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/usecaseutil"
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
		return usecaseutil.Wrap(err, "CleanupUseCase - CleanupDeletedTeams")
	}
	return nil
}
