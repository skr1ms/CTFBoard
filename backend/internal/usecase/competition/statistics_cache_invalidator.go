package competition

import (
	"context"

	"github.com/wahrwelt-kit/go-cachekit"
)

const statsCachePrefix = "stats:"

type StatsCacheInvalidatorImpl struct {
	Cache *cachekit.Cache
}

func (s *StatsCacheInvalidatorImpl) InvalidateStatistics(ctx context.Context) error {
	if s == nil || s.Cache == nil {
		return nil
	}

	return s.Cache.DeleteByPrefix(ctx, statsCachePrefix)
}
