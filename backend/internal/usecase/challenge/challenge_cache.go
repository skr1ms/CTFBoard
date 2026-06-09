package challenge

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

func (uc *ChallengeUseCase) InvalidateScoreboardCache(ctx context.Context) {
	if uc.deps.ScoreboardCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	})
}

func (uc *ChallengeUseCase) InvalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	})
}

func (uc *ChallengeUseCase) invalidateStatisticsCache(ctx context.Context, op string) {
	if uc.deps.StatsCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if err := uc.deps.StatsCache.InvalidateStatistics(ctx); err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - " + op + ": failed to invalidate statistics cache")
		}
	})
}

// InvalidateChallengeListCache evicts all challenge list cache entries by
// deleting the shared base list (challengeBaseCachePrefix), the per-team
// solved-set (challengeSolvedCachePrefix), and cached requirement pairs. Call
// this after any write that affects the challenge catalogue visible to all teams
// (e.g. create, delete, publish).
func (uc *ChallengeUseCase) InvalidateChallengeListCache(ctx context.Context) {
	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		uc.requirementPairsSF.Forget(requirementPairsCacheKey)

		if uc.deps.ListCache == nil {
			return
		}

		err := uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix base")
		}

		err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeSolvedCachePrefix)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix solved")
		}

		err = uc.deps.ListCache.Del(ctx, requirementPairsCacheKey)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - Del requirement pairs")
		}
	})
}

// InvalidateChallengeListCacheForTeam evicts only the cache entries scoped to
// a specific team: the team's solved-set key. The shared base list
// (challengeBaseCachePrefix) is also deleted unless the competition freeze is
// currently active - during a freeze other teams still rely on the frozen base
// view, so it must not be evicted by a single-team event.
func (uc *ChallengeUseCase) InvalidateChallengeListCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if uc.deps.ListCache == nil {
			return
		}

		err := uc.deps.ListCache.Del(ctx, challengeSolvedCachePrefix+teamID.String())
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - Del solved")
		}

		comp := computil.Cached(ctx, uc.deps.CompUC, uc.deps.CompRepo)
		if comp != nil && comp.IsFreezeActive() {
			return
		}

		err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)
		if err != nil {
			uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - DeleteByPrefix base")
		}
	})
}

func (uc *ChallengeUseCase) InvalidateAll(ctx context.Context) { uc.InvalidateChallengeListCache(ctx) }

func (uc *ChallengeUseCase) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}
