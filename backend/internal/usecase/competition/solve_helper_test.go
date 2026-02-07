package competition

import (
	"github.com/go-redis/redismock/v9"
	"github.com/skr1ms/CTFBoard/pkg/cache"
)

func (h *CompetitionTestHelper) CreateSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSolveUseCase(SolveDeps{
		SolveRepo:       h.deps.solveRepo,
		ChallengeRepo:   h.deps.challengeRepo,
		CompetitionRepo: h.deps.competitionRepo,
		UserRepo:        h.deps.userRepo,
		TeamRepo:        h.deps.teamRepo,
		TxRepo:          h.deps.txRepo,
		Cache:           cache.New(client),
		ScoreboardCache: nil,
		Broadcaster:     nil,
	}), redis
}
