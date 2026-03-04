package competition

import (
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/go-redis/redismock/v9"
)

func (h *CompetitionTestHelper) CreateSolveUseCase() (*SolveUseCase, redismock.ClientMock) {
	h.t.Helper()
	client, redis := redismock.NewClientMock()
	return NewSolveUseCase(SolveDeps{
		SolveRepo:          h.deps.solveRepo,
		ChallengeRepo:      h.deps.challengeRepo,
		CompetitionRepo:    h.deps.competitionRepo,
		UserRepo:           h.deps.userRepo,
		TeamRepo:           h.deps.teamRepo,
		TM:                 h.deps.tm,
		Cache:              cache.New(client),
		ScoreboardCache:    nil,
		ChallengeListCache: nil,
		Broadcaster:        nil,
	}), redis
}
