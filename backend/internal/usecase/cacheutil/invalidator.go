package cacheutil

import (
	"context"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-logkit"
)

// InvalidateUser removes the cached record for userID when userCache is non-nil.
func InvalidateUser(ctx context.Context, userCache UserCacheInvalidator, userID uuid.UUID) {
	if userCache != nil {
		userCache.InvalidateUser(ctx, userID)
	}
}

// InvalidateScoreboard purges all scoreboard cache entries when sbCache is non-nil.
func InvalidateScoreboard(ctx context.Context, sbCache ScoreboardCacheInvalidator) {
	if sbCache != nil {
		sbCache.InvalidateAll(ctx)
	}
}

// InvalidateScoreboardForTeam purges scoreboard entries for a specific team when sbCache is non-nil.
func InvalidateScoreboardForTeam(ctx context.Context, sbCache ScoreboardCacheInvalidator, teamID uuid.UUID) {
	if sbCache != nil {
		sbCache.InvalidateForTeam(ctx, teamID)
	}
}

// InvalidateTeam removes the cached team record from the raw cache when teamCache is non-nil.
// Logs a warning on deletion error using the provided logger.
func InvalidateTeam(ctx context.Context, teamCache *cachekit.Cache, logger logkit.Logger, teamID uuid.UUID) {
	if teamCache == nil {
		return
	}

	if err := teamCache.Del(ctx, KeyTeam(teamID.String())); err != nil {
		logger.WithError(err).Warn("cacheutil - InvalidateTeam - Del")
	}
}

// InvalidateChallengeList purges all challenge list cache entries when clCache is non-nil.
func InvalidateChallengeList(ctx context.Context, clCache ChallengeListCacheInvalidator) {
	if clCache != nil {
		clCache.InvalidateAll(ctx)
	}
}
