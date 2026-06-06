package cache

import (
	"context"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase/cacheutil"
)

// TeamBracketIDGetter resolves the bracket a team belongs to, used for targeted cache invalidation.
type TeamBracketIDGetter interface {
	GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error)
}

type ScoreboardCacheInvalidator = cacheutil.ScoreboardCacheInvalidator

type UserCacheInvalidator = cacheutil.UserCacheInvalidator

// UserCacheService manages cache invalidation for individual user records.
type UserCacheService struct {
	cache *cachekit.Cache
}

// NewUserCacheService creates a UserCacheService backed by the given cache instance.
func NewUserCacheService(c *cachekit.Cache) *UserCacheService {
	return &UserCacheService{cache: c}
}

// InvalidateUser removes the cached user record for userID from Redis.
func (s *UserCacheService) InvalidateUser(ctx context.Context, userID uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}

	_ = s.cache.Del(ctx, cacheutil.KeyUser(userID.String()))
}

var _ cacheutil.UserCacheInvalidator = (*UserCacheService)(nil)

type ChallengeListCacheInvalidator = cacheutil.ChallengeListCacheInvalidator

// ScoreboardCacheService manages invalidation of scoreboard cache entries in both Redis and an
// optional in-process local cache, with support for bracket-scoped and live-only invalidation.
type ScoreboardCacheService struct {
	cache              *cachekit.Cache
	getter             TeamBracketIDGetter
	localClear         func()
	localClearLiveOnly func(keys []string)
}

// NewScoreboardCacheService creates a ScoreboardCacheService backed by the given Redis cache and bracket getter.
func NewScoreboardCacheService(c *cachekit.Cache, getter TeamBracketIDGetter) *ScoreboardCacheService {
	return &ScoreboardCacheService{cache: c, getter: getter}
}

// RegisterLocalCache registers a callback that clears the entire in-process scoreboard cache.
func (s *ScoreboardCacheService) RegisterLocalCache(fn func()) {
	s.localClear = fn
}

// RegisterLocalCacheLiveOnly registers a callback that clears only the specified live scoreboard keys
// from the in-process cache.
func (s *ScoreboardCacheService) RegisterLocalCacheLiveOnly(fn func(keys []string)) {
	s.localClearLiveOnly = fn
}

// InvalidateAll purges all scoreboard cache entries (live, frozen, and per-bracket) from both the
// local in-process cache and Redis.
func (s *ScoreboardCacheService) InvalidateAll(ctx context.Context) {
	if s == nil {
		return
	}

	if s.localClear != nil {
		s.localClear()
	}

	if s.cache != nil {
		_ = s.cache.Del(ctx, cacheutil.KeyScoreboard, cacheutil.KeyScoreboardFrozen)
		_ = s.cache.DeleteByPrefix(ctx, cacheutil.KeyScoreboardFrozenPrefix)
		_ = s.cache.DeleteByPrefix(ctx, cacheutil.KeyScoreboardBracketPrefix)
	}
}

// InvalidateForTeam purges all scoreboard cache entries. Currently equivalent to InvalidateAll
// because any team score change can affect global rankings.
func (s *ScoreboardCacheService) InvalidateForTeam(ctx context.Context, _ uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}

	s.InvalidateAll(ctx)
}

// InvalidateLiveOnly purges only the live (non-frozen) scoreboard keys for the global board and,
// when resolvable, the team's bracket board. Frozen snapshots are left intact.
func (s *ScoreboardCacheService) InvalidateLiveOnly(ctx context.Context, teamID uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}

	liveKeys := []string{cacheutil.KeyScoreboard}

	if s.getter != nil {
		if bracketID, err := s.getter.GetTeamBracketID(ctx, teamID); err == nil && bracketID != nil {
			liveKeys = append(liveKeys, cacheutil.KeyScoreboardBracket(bracketID.String()))
		}
	}

	if s.localClearLiveOnly != nil {
		s.localClearLiveOnly(liveKeys)
	} else if s.localClear != nil {
		s.localClear()
	}

	err := s.cache.Del(ctx, liveKeys...)
	if err != nil {
		return
	}
}

var _ cacheutil.ScoreboardCacheInvalidator = (*ScoreboardCacheService)(nil)

// InvalidateWithFreezeAwareness invalidates scoreboard cache entries respecting freeze state.
// When frozen, only the live scoreboard is invalidated to preserve the frozen snapshot.
// When not frozen, the full team-scoped invalidation is performed.
func InvalidateWithFreezeAwareness(ctx context.Context, cache ScoreboardCacheInvalidator, teamID uuid.UUID, frozen bool) {
	cacheutil.InvalidateWithFreezeAwareness(ctx, cache, teamID, frozen)
}
