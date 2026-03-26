package challenge

import (
	"context"

	"github.com/google/uuid"
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

func (uc *ChallengeUseCase) InvalidateChallengeListCache(ctx context.Context) {
	if uc.deps.ListCache == nil {
		return
	}

	err := uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix)
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix list")
	}

	err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix base")
	}

	err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeSolvedCachePrefix)
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCache - DeleteByPrefix solved")
	}
}

func (uc *ChallengeUseCase) InvalidateChallengeListCacheForTeam(ctx context.Context, teamID uuid.UUID) {
	if uc.deps.ListCache == nil {
		return
	}

	err := uc.deps.ListCache.DeleteByPrefix(ctx, challengeListCachePrefix+teamID.String()+":")
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - DeleteByPrefix list")
	}

	err = uc.deps.ListCache.Del(ctx, challengeSolvedCachePrefix+teamID.String())
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - Del solved")
	}

	comp := uc.getCompetitionForGetAll(ctx)
	if comp != nil && comp.IsFreezeActive() {
		return
	}

	err = uc.deps.ListCache.DeleteByPrefix(ctx, challengeBaseCachePrefix)
	if err != nil && uc.deps.Logger != nil {
		uc.deps.Logger.WithError(err).Warn("ChallengeUseCase - InvalidateChallengeListCacheForTeam - DeleteByPrefix base")
	}
}

func (uc *ChallengeUseCase) InvalidateAll(ctx context.Context) { uc.InvalidateChallengeListCache(ctx) }

func (uc *ChallengeUseCase) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}
