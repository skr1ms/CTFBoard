package challenge

import (
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
)

func (h *ChallengeTestHelper) CreateHintUseCase() (*HintUseCase, redismock.ClientMock) {
	h.t.Helper()
	_, redis := redismock.NewClientMock()
	return NewHintUseCase(HintDeps{
		HintRepo: h.deps.hintRepo, HintUnlockRepo: h.deps.hintUnlockRepo, AwardRepo: h.deps.awardRepo,
		TxRepo: h.deps.txRepo, SolveRepo: h.deps.solveRepo, ScoreboardCache: nil,
	}), redis
}

func (h *ChallengeTestHelper) NewHint(id, challengeID uuid.UUID, content string, cost, orderIndex int) *entity.Hint {
	h.t.Helper()
	return &entity.Hint{
		ID:          id,
		ChallengeID: challengeID,
		Content:     content,
		Cost:        cost,
		OrderIndex:  orderIndex,
	}
}
