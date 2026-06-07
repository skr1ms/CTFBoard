package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromHintWithUnlock(hw *usecase.HintWithUnlockStatus) openapi.HintResponse {
	res := openapi.HintResponse{
		ID:         new(hw.Hint.ID.String()),
		Title:      new(hw.Hint.Title),
		Cost:       new(hw.Hint.Cost),
		OrderIndex: new(hw.Hint.OrderIndex),
		Unlocked:   new(hw.Unlocked),
	}
	if hw.Unlocked {
		res.Content = new(hw.Hint.Content)
	}

	return res
}

func FromHintWithUnlockList(hints []*usecase.HintWithUnlockStatus) []openapi.HintResponse {
	return lo.Map(hints, func(h *usecase.HintWithUnlockStatus, _ int) openapi.HintResponse { return FromHintWithUnlock(h) })
}

func FromUnlockedHint(h *domain.Hint) openapi.HintResponse {
	return openapi.HintResponse{
		ID:         new(h.ID.String()),
		Title:      new(h.Title),
		Cost:       new(h.Cost),
		OrderIndex: new(h.OrderIndex),
		Content:    new(h.Content),
		Unlocked:   new(true),
	}
}

func FromHint(h *domain.Hint) openapi.HintAdminResponse {
	return openapi.HintAdminResponse{
		ID:          new(h.ID.String()),
		ChallengeID: new(h.ChallengeID.String()),
		Title:       new(h.Title),
		Content:     new(h.Content),
		Cost:        new(h.Cost),
		OrderIndex:  new(h.OrderIndex),
	}
}

func FromUnlock(u *domain.UnlockWithDetails) openapi.UnlockResponse {
	t := u.UnlockedAt
	unlockType := openapi.UnlockResponseType(u.Type)

	return openapi.UnlockResponse{
		ID:          new(u.ID.String()),
		Type:        &unlockType,
		ResourceID:  new(u.ResourceID.String()),
		HintID:      new(u.HintID.String()),
		TeamID:      new(u.TeamID.String()),
		UnlockedAt:  &t,
		ChallengeID: new(u.ChallengeID.String()),
		HintCost:    new(u.HintCost),
	}
}

func FromUnlockList(items []*domain.UnlockWithDetails, total int64, page, perPage int) openapi.UnlockListResponse {
	data, meta := BuildListResponse(items, FromUnlock, total, page, perPage)

	return openapi.UnlockListResponse{Data: &data, Meta: meta}
}
