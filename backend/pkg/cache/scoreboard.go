package cache

import (
	"context"

	"github.com/google/uuid"
)

type TeamBracketIDGetter interface {
	GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error)
}

type ScoreboardCacheInvalidator interface {
	InvalidateAll(ctx context.Context)
	InvalidateForTeam(ctx context.Context, teamID uuid.UUID)
}

// UserCacheInvalidator evicts a cached user entry so the next request loads fresh data.
type UserCacheInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID)
}

// UserCacheService implements UserCacheInvalidator on top of Cache.
type UserCacheService struct {
	cache *Cache
}

func NewUserCacheService(c *Cache) *UserCacheService {
	return &UserCacheService{cache: c}
}

func (s *UserCacheService) InvalidateUser(ctx context.Context, userID uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}
	_ = s.cache.Del(ctx, KeyUser(userID.String())) //nolint:errcheck // best-effort invalidation
}

var _ UserCacheInvalidator = (*UserCacheService)(nil)

// ChallengeListCacheInvalidator invalidates challenge list cache (e.g. on solve create/delete).
type ChallengeListCacheInvalidator interface {
	InvalidateAll(ctx context.Context)
	InvalidateForTeam(ctx context.Context, teamID uuid.UUID)
}

type ScoreboardCacheService struct {
	cache      *Cache
	getter     TeamBracketIDGetter
	localClear func() // clears in-process scoreboard caches on every invalidation
}

func NewScoreboardCacheService(c *Cache, getter TeamBracketIDGetter) *ScoreboardCacheService {
	return &ScoreboardCacheService{cache: c, getter: getter}
}

// RegisterLocalCache registers a callback that is invoked on every cache
// invalidation to also clear in-process (non-Redis) scoreboard caches.
// Must be called before the service is used.
func (s *ScoreboardCacheService) RegisterLocalCache(fn func()) {
	s.localClear = fn
}

func (s *ScoreboardCacheService) InvalidateAll(ctx context.Context) {
	if s == nil {
		return
	}
	if s.localClear != nil {
		s.localClear()
	}
	if s.cache != nil {
		_ = s.cache.Del(ctx, KeyScoreboard, KeyScoreboardFrozen) //nolint:errcheck // best-effort invalidation
	}
}

func (s *ScoreboardCacheService) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}
	s.InvalidateAll(ctx) // also calls localClear
	if s.getter == nil {
		return
	}
	bracketID, err := s.getter.GetTeamBracketID(ctx, teamID)
	if err != nil || bracketID == nil {
		return
	}
	idStr := bracketID.String()
	_ = s.cache.Del(ctx, KeyScoreboardBracket(idStr), KeyScoreboardBracketFrozen(idStr)) //nolint:errcheck // best-effort invalidation
}

var _ ScoreboardCacheInvalidator = (*ScoreboardCacheService)(nil)
