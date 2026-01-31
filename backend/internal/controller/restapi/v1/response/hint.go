package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/openapi"
	"github.com/skr1ms/CTFBoard/internal/usecase/challenge"
)

// FromHintWithUnlock creates HintResponse from HintWithUnlock entity
func FromHintWithUnlock(hw *challenge.HintWithUnlockStatus) openapi.ResponseHintResponse {
	res := openapi.ResponseHintResponse{
		ID:         ptr(hw.Hint.ID.String()),
		Cost:       ptr(hw.Hint.Cost),
		OrderIndex: ptr(hw.Hint.OrderIndex),
		Unlocked:   ptr(hw.Unlocked),
	}
	if hw.Unlocked {
		res.Content = ptr(hw.Hint.Content)
	}
	return res
}

// FromHintWithUnlockList creates a list of HintResponse
func FromHintWithUnlockList(hints []*challenge.HintWithUnlockStatus) []openapi.ResponseHintResponse {
	res := make([]openapi.ResponseHintResponse, len(hints))
	for i, h := range hints {
		res[i] = FromHintWithUnlock(h)
	}
	return res
}

// FromUnlockedHint creates HintResponse from unlocked Hint entity
func FromUnlockedHint(h *entity.Hint) openapi.ResponseHintResponse {
	return openapi.ResponseHintResponse{
		ID:         ptr(h.ID.String()),
		Cost:       ptr(h.Cost),
		OrderIndex: ptr(h.OrderIndex),
		Content:    ptr(h.Content),
		Unlocked:   ptr(true),
	}
}

func FromHint(h *entity.Hint) openapi.ResponseHintAdminResponse {
	return openapi.ResponseHintAdminResponse{
		ID:          ptr(h.ID.String()),
		ChallengeID: ptr(h.ChallengeID.String()),
		Content:     ptr(h.Content),
		Cost:        ptr(h.Cost),
		OrderIndex:  ptr(h.OrderIndex),
	}
}
