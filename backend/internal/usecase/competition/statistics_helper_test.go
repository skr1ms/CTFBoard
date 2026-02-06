package competition

import (
	"github.com/go-redis/redismock/v9"
	"github.com/skr1ms/CTFBoard/pkg/cache"
)

func (h *CompetitionTestHelper) CreateStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, mock := redismock.NewClientMock()
	return NewStatisticsUseCase(h.deps.statsRepo, cache.New(client)), mock
}
