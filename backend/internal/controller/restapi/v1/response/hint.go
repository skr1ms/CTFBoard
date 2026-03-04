package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromHintWithUnlock(hw *usecase.HintWithUnlockStatus) openapi.HintResponse {
	res := openapi.HintResponse{
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

func FromHintWithUnlockList(hints []*usecase.HintWithUnlockStatus) []openapi.HintResponse {
	res := make([]openapi.HintResponse, len(hints))
	for i, h := range hints {
		res[i] = FromHintWithUnlock(h)
	}
	return res
}

func FromUnlockedHint(h *entity.Hint) openapi.HintResponse {
	return openapi.HintResponse{
		ID:         ptr(h.ID.String()),
		Cost:       ptr(h.Cost),
		OrderIndex: ptr(h.OrderIndex),
		Content:    ptr(h.Content),
		Unlocked:   ptr(true),
	}
}

func FromHint(h *entity.Hint) openapi.HintAdminResponse {
	return openapi.HintAdminResponse{
		ID:          ptr(h.ID.String()),
		ChallengeID: ptr(h.ChallengeID.String()),
		Content:     ptr(h.Content),
		Cost:        ptr(h.Cost),
		OrderIndex:  ptr(h.OrderIndex),
	}
}

func FromHintUnlock(u *entity.HintUnlockWithDetails) openapi.HintUnlockResponse {
	t := u.UnlockedAt
	return openapi.HintUnlockResponse{
		ID:          ptr(u.ID.String()),
		HintID:      ptr(u.HintID.String()),
		TeamID:      ptr(u.TeamID.String()),
		UnlockedAt:  &t,
		ChallengeID: ptr(u.ChallengeID.String()),
		HintCost:    ptr(u.HintCost),
	}
}

func FromHintUnlockList(items []*entity.HintUnlockWithDetails, total int64, page, perPage int) openapi.HintUnlockListResponse {
	res := make([]openapi.HintUnlockResponse, len(items))
	for i, item := range items {
		res[i] = FromHintUnlock(item)
	}
	return openapi.HintUnlockListResponse{
		Data: &res,
		Meta: &openapi.PaginationMeta{
			Page:       ptr(page),
			PerPage:    ptr(perPage),
			Total:      ptr(int(total)),
			TotalPages: ptr(TotalPages(total, perPage)),
		},
	}
}
