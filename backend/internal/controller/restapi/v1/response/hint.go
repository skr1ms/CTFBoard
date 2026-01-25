package response

import (
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase"
)

type HintResponse struct {
	Id         string  `json:"id"`
	Cost       int     `json:"cost"`
	OrderIndex int     `json:"order_index"`
	Content    *string `json:"content,omitempty"`
	Unlocked   bool    `json:"unlocked"`
}

// FromHintWithUnlock creates HintResponse from HintWithUnlock entity
func FromHintWithUnlock(hw *usecase.HintWithUnlockStatus) HintResponse {
	res := HintResponse{
		Id:         hw.Hint.Id.String(),
		Cost:       hw.Hint.Cost,
		OrderIndex: hw.Hint.OrderIndex,
		Unlocked:   hw.Unlocked,
	}
	if hw.Unlocked {
		res.Content = &hw.Hint.Content
	}
	return res
}

// FromHintWithUnlockList creates a list of HintResponse
func FromHintWithUnlockList(hints []*usecase.HintWithUnlockStatus) []HintResponse {
	res := make([]HintResponse, len(hints))
	for i, h := range hints {
		res[i] = FromHintWithUnlock(h)
	}
	return res
}

// FromUnlockedHint creates HintResponse from unlocked Hint entity
func FromUnlockedHint(h *entity.Hint) HintResponse {
	return HintResponse{
		Id:         h.Id.String(),
		Cost:       h.Cost,
		OrderIndex: h.OrderIndex,
		Content:    &h.Content,
		Unlocked:   true,
	}
}

type HintAdminResponse struct {
	Id          string `json:"id"`
	ChallengeId string `json:"challenge_id"`
	Content     string `json:"content"`
	Cost        int    `json:"cost"`
	OrderIndex  int    `json:"order_index"`
}

func FromHint(h *entity.Hint) HintAdminResponse {
	return HintAdminResponse{
		Id:          h.Id.String(),
		ChallengeId: h.ChallengeId.String(),
		Content:     h.Content,
		Cost:        h.Cost,
		OrderIndex:  h.OrderIndex,
	}
}
