package cacheutil

import (
	"context"

	"github.com/google/uuid"
)

// ScoreboardCacheInvalidator is the usecase-owned port for scoreboard cache eviction.
type ScoreboardCacheInvalidator interface {
	InvalidateAll(ctx context.Context)
	InvalidateForTeam(ctx context.Context, teamID uuid.UUID)
	InvalidateLiveOnly(ctx context.Context, teamID uuid.UUID)
}

// UserCacheInvalidator is the usecase-owned port for user cache eviction.
type UserCacheInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID)
}

// ChallengeListCacheInvalidator is the usecase-owned port for challenge list cache eviction.
type ChallengeListCacheInvalidator interface {
	InvalidateAll(ctx context.Context)
	InvalidateForTeam(ctx context.Context, teamID uuid.UUID)
}

// InvalidateWithFreezeAwareness invalidates scoreboard cache entries respecting freeze state.
func InvalidateWithFreezeAwareness(ctx context.Context, cache ScoreboardCacheInvalidator, teamID uuid.UUID, frozen bool) {
	if cache == nil {
		return
	}

	if frozen {
		cache.InvalidateLiveOnly(ctx, teamID)

		return
	}

	cache.InvalidateForTeam(ctx, teamID)
}
