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

type ScoreboardCacheService struct {
	cache  *Cache
	getter TeamBracketIDGetter
}

func NewScoreboardCacheService(c *Cache, getter TeamBracketIDGetter) *ScoreboardCacheService {
	return &ScoreboardCacheService{cache: c, getter: getter}
}

func (s *ScoreboardCacheService) InvalidateAll(ctx context.Context) {
	if s != nil && s.cache != nil {
		s.cache.Del(ctx, KeyScoreboard, KeyScoreboardFrozen)
	}
}

func (s *ScoreboardCacheService) InvalidateForTeam(ctx context.Context, teamID uuid.UUID) {
	s.InvalidateAll(ctx)
	if s.getter == nil {
		return
	}
	bracketID, err := s.getter.GetTeamBracketID(ctx, teamID)
	if err != nil || bracketID == nil {
		return
	}
	idStr := bracketID.String()
	s.cache.Del(ctx, KeyScoreboardBracket(idStr), KeyScoreboardBracketFrozen(idStr))
}

var _ ScoreboardCacheInvalidator = (*ScoreboardCacheService)(nil)
