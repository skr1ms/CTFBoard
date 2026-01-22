package usecase

import (
	"context"
	"time"

	"github.com/skr1ms/CTFBoard/internal/repo"
)

type CleanupUseCase struct {
	repo repo.TeamRepository
}

func NewCleanupUseCase(repo repo.TeamRepository) *CleanupUseCase {
	return &CleanupUseCase{repo: repo}
}

func (uc *CleanupUseCase) CleanupDeletedTeams(ctx context.Context, olderThan time.Duration) error {
	cutoffDate := time.Now().Add(-olderThan)
	return uc.repo.HardDeleteTeams(ctx, cutoffDate)
}
