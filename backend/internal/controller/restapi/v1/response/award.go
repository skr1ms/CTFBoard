package response

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAward(a *entity.Award) openapi.AwardResponse {
	res := openapi.AwardResponse{
		ID:          ptr(a.ID.String()),
		TeamID:      ptr(a.TeamID.String()),
		Value:       ptr(a.Value),
		Description: ptr(a.Description),
		CreatedAt:   ptr(a.CreatedAt),
	}
	if a.CreatedBy != nil {
		res.CreatedBy = ptr(a.CreatedBy.String())
	}
	return res
}

func FromAwardList(items []*entity.Award) []openapi.AwardResponse {
	res := make([]openapi.AwardResponse, len(items))
	for i, item := range items {
		res[i] = FromAward(item)
	}
	return res
}
