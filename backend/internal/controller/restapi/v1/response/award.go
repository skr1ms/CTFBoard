package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromAward(a *domain.Award) openapi.AwardResponse {
	res := openapi.AwardResponse{
		ID:          httputil.Ptr(a.ID.String()),
		TeamID:      httputil.Ptr(a.TeamID.String()),
		Value:       httputil.Ptr(a.Value),
		Description: httputil.Ptr(a.Description),
		CreatedAt:   httputil.Ptr(a.CreatedAt),
	}
	if a.CreatedBy != nil {
		res.CreatedBy = httputil.Ptr(a.CreatedBy.String())
	}
	return res
}

func FromAwardList(items []*domain.Award) []openapi.AwardResponse {
	return lo.Map(items, func(item *domain.Award, _ int) openapi.AwardResponse { return FromAward(item) })
}
