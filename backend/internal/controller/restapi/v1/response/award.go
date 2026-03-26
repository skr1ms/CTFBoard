package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAward(a *domain.Award) openapi.AwardResponse {
	res := openapi.AwardResponse{
		ID:          new(a.ID.String()),
		TeamID:      new(a.TeamID.String()),
		Value:       new(a.Value),
		Description: new(a.Description),
		CreatedAt:   new(a.CreatedAt),
	}
	if a.CreatedBy != nil {
		res.CreatedBy = new(a.CreatedBy.String())
	}

	return res
}

func FromAwardList(items []*domain.Award) []openapi.AwardResponse {
	return lo.Map(items, func(item *domain.Award, _ int) openapi.AwardResponse { return FromAward(item) })
}
