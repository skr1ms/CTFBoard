package wire

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/repo"
	iws "github.com/TakuyaYagam1/AstroCTFb/internal/websocket"
)

func (g *teamBracketIDGetter) GetTeamBracketID(ctx context.Context, teamID uuid.UUID) (*uuid.UUID, error) {
	team, err := g.r.GetByID(ctx, teamID)
	if err != nil {
		return nil, fmt.Errorf("teamBracketIDGetter - GetTeamBracketID - TeamRepo.GetByID: %w", err)
	}

	if team == nil {
		return nil, fmt.Errorf("teamBracketIDGetter - GetTeamBracketID: team %s not found", teamID)
	}

	return team.BracketID, nil
}

func ProvideScoreboardCacheService(c *cachekit.Cache, teamRepo repo.TeamRepository) *cache.ScoreboardCacheService {
	return cache.NewScoreboardCacheService(c, &teamBracketIDGetter{r: teamRepo})
}

func ProvideUserCacheService(c *cachekit.Cache) *cache.UserCacheService {
	return cache.NewUserCacheService(c)
}

func ProvideBroadcaster(ctx context.Context, hub *wskit.Hub) *iws.Broadcaster {
	return iws.NewBroadcaster(ctx, hub)
}

func ProvideCache(r *redis.Client) *cachekit.Cache {
	return cachekit.New(r)
}

func ProvideKeyValueStore(r *redis.Client) cachekit.KeyValueStore {
	return &cachekit.RedisKeyValueStore{Client: r}
}

func ProvidePubSubStore(r *redis.Client) cachekit.PubSubStore {
	return &cachekit.RedisPubSubStore{Client: r}
}
