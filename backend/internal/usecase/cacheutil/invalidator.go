package cacheutil

import (
	"context"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/txctx"
)

// InvalidateUser removes the cached record for userID when userCache is non-nil.
func InvalidateUser(ctx context.Context, userCache UserCacheInvalidator, userID uuid.UUID) {
	if userCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		userCache.InvalidateUser(ctx, userID)
	})
}

// InvalidateScoreboard purges all scoreboard cache entries when sbCache is non-nil.
func InvalidateScoreboard(ctx context.Context, sbCache ScoreboardCacheInvalidator) {
	if sbCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		sbCache.InvalidateAll(ctx)
	})
}

// InvalidateScoreboardForTeam purges scoreboard entries for a specific team when sbCache is non-nil.
func InvalidateScoreboardForTeam(ctx context.Context, sbCache ScoreboardCacheInvalidator, teamID uuid.UUID) {
	if sbCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		sbCache.InvalidateForTeam(ctx, teamID)
	})
}

// InvalidateTeam removes the cached team record from the raw cache when teamCache is non-nil.
// Logs a warning on deletion error using the provided logger.
func InvalidateTeam(ctx context.Context, teamCache *cachekit.Cache, logger logkit.Logger, teamID uuid.UUID) {
	if teamCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if err := teamCache.Del(ctx, KeyTeam(teamID.String())); err != nil {
			logger.WithError(err).Warn("cacheutil - InvalidateTeam - Del")
		}
	})
}

// InvalidateChallengeList purges all challenge list cache entries when clCache is non-nil.
func InvalidateChallengeList(ctx context.Context, clCache ChallengeListCacheInvalidator) {
	if clCache == nil {
		return
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		clCache.InvalidateAll(ctx)
	})
}

func InvalidateStatistics(ctx context.Context, statsCache StatisticsCacheInvalidator, logger logkit.Logger, op string) {
	if statsCache == nil {
		return
	}

	if logger == nil {
		logger = logkit.Noop()
	}

	txctx.AfterCommitOrNow(ctx, func(ctx context.Context) {
		if err := statsCache.InvalidateStatistics(ctx); err != nil {
			logger.WithError(err).Warn(op + " - InvalidateStatistics")
		}
	})
}
