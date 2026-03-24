package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

func FromHintWithUnlock(hw *usecase.HintWithUnlockStatus) openapi.HintResponse {
	res := openapi.HintResponse{
		ID:         httputil.Ptr(hw.Hint.ID.String()),
		Title:      httputil.Ptr(hw.Hint.Title),
		Cost:       httputil.Ptr(hw.Hint.Cost),
		OrderIndex: httputil.Ptr(hw.Hint.OrderIndex),
		Unlocked:   httputil.Ptr(hw.Unlocked),
	}
	if hw.Unlocked {
		res.Content = httputil.Ptr(hw.Hint.Content)
	}
	return res
}

func FromHintWithUnlockList(hints []*usecase.HintWithUnlockStatus) []openapi.HintResponse {
	return lo.Map(hints, func(h *usecase.HintWithUnlockStatus, _ int) openapi.HintResponse { return FromHintWithUnlock(h) })
}

func FromUnlockedHint(h *domain.Hint) openapi.HintResponse {
	return openapi.HintResponse{
		ID:         httputil.Ptr(h.ID.String()),
		Title:      httputil.Ptr(h.Title),
		Cost:       httputil.Ptr(h.Cost),
		OrderIndex: httputil.Ptr(h.OrderIndex),
		Content:    httputil.Ptr(h.Content),
		Unlocked:   httputil.Ptr(true),
	}
}

func FromHint(h *domain.Hint) openapi.HintAdminResponse {
	return openapi.HintAdminResponse{
		ID:          httputil.Ptr(h.ID.String()),
		ChallengeID: httputil.Ptr(h.ChallengeID.String()),
		Title:       httputil.Ptr(h.Title),
		Content:     httputil.Ptr(h.Content),
		Cost:        httputil.Ptr(h.Cost),
		OrderIndex:  httputil.Ptr(h.OrderIndex),
	}
}

func FromHintUnlock(u *domain.HintUnlockWithDetails) openapi.HintUnlockResponse {
	t := u.UnlockedAt
	return openapi.HintUnlockResponse{
		ID:          httputil.Ptr(u.ID.String()),
		HintID:      httputil.Ptr(u.HintID.String()),
		TeamID:      httputil.Ptr(u.TeamID.String()),
		UnlockedAt:  &t,
		ChallengeID: httputil.Ptr(u.ChallengeID.String()),
		HintCost:    httputil.Ptr(u.HintCost),
	}
}

func FromHintUnlockList(items []*domain.HintUnlockWithDetails, total int64, page, perPage int) openapi.HintUnlockListResponse {
	data, meta := BuildListResponse(items, FromHintUnlock, total, page, perPage)
	return openapi.HintUnlockListResponse{Data: &data, Meta: meta}
}
