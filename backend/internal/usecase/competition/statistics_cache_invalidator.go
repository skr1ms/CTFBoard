package competition

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
)

const (
	statsCachePrefix       = "stats:"
	statsFunnelCachePrefix = "stats:funnel:"
)

type StatsCacheInvalidatorImpl struct {
	Cache *cachekit.Cache
}

func (s *StatsCacheInvalidatorImpl) InvalidateStatistics(ctx context.Context) error {
	if s == nil || s.Cache == nil {
		return nil
	}

	return s.Cache.DeleteByPrefix(ctx, statsCachePrefix)
}

type FunnelStatsCacheInvalidatorImpl struct {
	Cache *cachekit.Cache
}

func (s *FunnelStatsCacheInvalidatorImpl) InvalidateStatistics(ctx context.Context) error {
	if s == nil || s.Cache == nil {
		return nil
	}

	return s.Cache.DeleteByPrefix(ctx, statsFunnelCachePrefix)
}
