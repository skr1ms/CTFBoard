package competition

import (
	"github.com/go-redis/redismock/v9"
	"github.com/skr1ms/CTFBoard/pkg/cache"
)

func (h *CompetitionTestHelper) CreateSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSolveUseCase(
		h.deps.solveRepo,
		h.deps.challengeRepo,
		h.deps.competitionRepo,
		h.deps.userRepo,
		h.deps.teamRepo,
		h.deps.txRepo,
		cache.New(client),
		nil,
	), redis
}
