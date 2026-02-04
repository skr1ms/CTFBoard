package competition

import (
	"github.com/go-redis/redismock/v9"
)

func (h *CompetitionTestHelper) CreateStatisticsUseCase() (*StatisticsUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewStatisticsUseCase(h.deps.statsRepo, client), redis
}
