package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromBanAppeal(a *domain.BanAppeal) openapi.BanAppealResponse {
	if a == nil {
		return openapi.BanAppealResponse{}
	}

	resp := openapi.BanAppealResponse{
		ID:            new(a.ID.String()),
		UserID:        new(a.UserID.String()),
		Message:       new(a.Message),
		Decision:      new(string(a.Decision)),
		CreatedAt:     timePtr(&a.CreatedAt),
		ReviewedAt:    timePtr(a.ReviewedAt),
		AdminResponse: a.AdminResponse,
	}

	return resp
}

func FromBanAppealList(appeals []*domain.BanAppeal, total int64, page, perPage int) openapi.BanAppealListResponse {
	items, meta := BuildListResponse(appeals, FromBanAppeal, total, page, perPage)

	return openapi.BanAppealListResponse{
		Appeals: &items,
		Meta:    meta,
	}
}
