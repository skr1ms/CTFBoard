package competition

import (
	"context"

	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

func (uc *SolveUseCase) invalidateStatisticsCache(ctx context.Context, op string) {
	if uc == nil || uc.deps.StatsCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if err := uc.deps.StatsCache.InvalidateStatistics(ctx); err != nil {
			uc.deps.Logger.WithError(err).Warn("SolveUseCase - " + op + ": failed to invalidate statistics cache")
		}
	})
}

func (uc *SubmissionUseCase) invalidateStatisticsCache(ctx context.Context, op string) {
	if uc == nil || uc.deps.StatsCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if err := uc.deps.StatsCache.InvalidateStatistics(ctx); err != nil {
			uc.deps.Logger.WithError(err).Warn("SubmissionUseCase - " + op + ": failed to invalidate statistics cache")
		}
	})
}
