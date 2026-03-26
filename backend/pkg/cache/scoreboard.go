package cache

import (
	"context"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
)

type TeamBracketIDGetter interface {
	GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error)
}

type ScoreboardCacheInvalidator interface {
	InvalidateAll(ctx context.Context)
	InvalidateForTeam(ctx context.Context, teamID uuid.UUID)
	InvalidateLiveOnly(ctx context.Context, teamID uuid.UUID)
}

type UserCacheInvalidator interface {
	InvalidateUser(ctx context.Context, userID uuid.UUID)
}

type UserCacheService struct {
	cache *cachekit.Cache
}

func NewUserCacheService(c *cachekit.Cache) *UserCacheService {
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
	cache              *cachekit.Cache
	getter             TeamBracketIDGetter
	localClear         func()
	localClearLiveOnly func(keys []string)
}

func NewScoreboardCacheService(c *cachekit.Cache, getter TeamBracketIDGetter) *ScoreboardCacheService {
	return &ScoreboardCacheService{cache: c, getter: getter}
}

func (s *ScoreboardCacheService) RegisterLocalCache(fn func()) {
	s.localClear = fn
}

func (s *ScoreboardCacheService) RegisterLocalCacheLiveOnly(fn func(keys []string)) {
	s.localClearLiveOnly = fn
}

func (s *ScoreboardCacheService) InvalidateAll(ctx context.Context) {
	if s == nil {
		return
	}

	if s.localClear != nil {
		s.localClear()
	}

	if s.cache != nil {
		_ = s.cache.Del(ctx, KeyScoreboard, KeyScoreboardFrozen)    //nolint:errcheck // best-effort invalidation
		_ = s.cache.DeleteByPrefix(ctx, KeyScoreboardFrozenPrefix)  //nolint:errcheck // best-effort invalidation
		_ = s.cache.DeleteByPrefix(ctx, KeyScoreboardBracketPrefix) //nolint:errcheck // best-effort invalidation
	}
}

func (s *ScoreboardCacheService) InvalidateForTeam(ctx context.Context, _ uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}

	s.InvalidateAll(ctx)
}

func (s *ScoreboardCacheService) InvalidateLiveOnly(ctx context.Context, teamID uuid.UUID) {
	if s == nil || s.cache == nil {
		return
	}

	liveKeys := []string{KeyScoreboard}

	if s.getter != nil {
		if bracketID, err := s.getter.GetTeamBracketID(ctx, teamID); err == nil && bracketID != nil {
			liveKeys = append(liveKeys, KeyScoreboardBracket(bracketID.String()))
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

var _ ScoreboardCacheInvalidator = (*ScoreboardCacheService)(nil)
