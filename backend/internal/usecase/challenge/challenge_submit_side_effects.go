package challenge

import (
	"context"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/computil"
)

// submitInvalidateCache invalidates scoreboard and challenge caches after a solve.
// Delegates to submitInvalidateCacheWithFrozenStatus after checking current freeze state.
func (uc *ChallengeUseCase) submitInvalidateCache(ctx context.Context, teamID uuid.UUID) {
	comp := computil.Fresh(ctx, uc.deps.CompRepo, uc.deps.CompUC)
	wasFrozen := comp != nil && comp.IsFreezeActive()
	uc.submitInvalidateCacheWithFrozenStatus(ctx, teamID, wasFrozen)
}

// submitInvalidateCacheWithFrozenStatus invalidates scoreboard and challenge list caches
// with awareness of freeze state. When frozen, only live scoreboard is invalidated
// (frozen snapshot is preserved). Challenge list cache is always invalidated.
func (uc *ChallengeUseCase) submitInvalidateCacheWithFrozenStatus(ctx context.Context, teamID uuid.UUID, wasFrozen bool) {
	cacheutil.InvalidateWithFreezeAwareness(ctx, uc.deps.ScoreboardCache, teamID, wasFrozen)
	uc.InvalidateChallengeListCacheForTeam(ctx, teamID)
}

// submitNotifySolve broadcasts a solve event (and optionally a first-blood event) to WebSocket/SSE clients.
// wasFrozen is forwarded from submitRecordSolve to skip the redundant CompRepo hit.
func (uc *ChallengeUseCase) submitNotifySolve(sc *submitContext, challenge *domain.Challenge, isFirstBlood, wasFrozen bool) {
	if uc.deps.Broadcaster == nil || challenge == nil || wasFrozen {
		return
	}

	uc.deps.Broadcaster.NotifySolve(sc.teamID, challenge.Title, challenge.Points, isFirstBlood)
}
