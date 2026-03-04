package competition

import (
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/go-redis/redismock/v9"
)

func (h *CompetitionTestHelper) CreateStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, mock := redismock.NewClientMock()
	return NewStatisticsUseCase(StatisticsDeps{StatsRepo: h.deps.statsRepo, Cache: cache.New(client)}), mock
}
