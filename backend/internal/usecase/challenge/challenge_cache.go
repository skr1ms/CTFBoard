package challenge

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

func (uc *ChallengeUseCase) InvalidateScoreboardCache(ctx context.Context) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateAll(ctx)
	}
}

func (uc *ChallengeUseCase) InvalidateScoreboardCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ScoreboardCache != nil {
		uc.deps.ScoreboardCache.InvalidateForTeam(ctx, teamID)
	}
}

func (uc *ChallengeUseCase) invalidateStatisticsCache(ctx context.Context, op string) {
	if uc.deps.StatsCache == nil {
		return
	}

	if err := uc.deps.StatsCache.InvalidateStatistics(ctx); err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - " + op + ": failed to invalidate statistics cache")
	}
}

// InvalidateChallengeListCache evicts all challenge list cache entries by
// deleting three key prefixes: the team-scoped paginated list (challengeListCachePrefix),
// the shared base list (challengeBaseCachePrefix), and the per-team solved-set
// (challengeSolvedCachePrefix). Call this after any write that affects the
// challenge catalogue visible to all teams (e.g. create, delete, publish).
func (uc *ChallengeUseCase) InvalidateChallengeListCache(ctx context.Context) {
	if uc.deps.ListCache == nil {
		return
	}

	err := uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix list")
	}

	err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix base")
	}

	err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeSolvedCachePrefix)
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix solved")
	}
}

// InvalidateChallengeListCacheForTeam evicts only the cache entries scoped to
// a specific team: the team-scoped paginated list prefix and the team's solved-set
// key. The shared base list (challengeBaseCachePrefix) is also deleted unless the
// competition freeze is currently active - during a freeze other teams still rely
// on the frozen base view, so it must not be evicted by a single-team event.
func (uc *ChallengeUseCase) InvalidateChallengeListCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ListCache == nil {
		return
	}

	err := uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix+teamID.String()+":")
	if err != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - DeleteByPrefix list")
	}

	err = uc.deps.ListCache.Del(ctx, challengeSolvedCachePrefix+teamID.String())
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
}

func (uc *ChallengeUseCase) InvalidateAll(ctx context.Context) { uc.InvalidateChallengeListCache(ctx) }

func (uc *ChallengeUseCase) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}
