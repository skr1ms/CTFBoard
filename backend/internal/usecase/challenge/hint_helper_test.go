package challenge

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
)

func (h *ChallengeTestHelper) CreateHintUseCase() (*HintUseCase, redismock.ClientMock) {
	h.t.Helper()
	_, redis := redismock.NewClientMock()
	return NewHintUseCase(HintDeps{
		HintRepo: h.deps.hintRepo, AwardRepo: h.deps.awardRepo,
		TM: h.deps.tm, SolveRepo: h.deps.solveRepo,
		CompRepo: h.deps.compRepo, TeamRepo: h.deps.teamRepo,
		ChallengeRepo:   h.deps.challengeRepo,
		ScoreboardCache: nil,
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
