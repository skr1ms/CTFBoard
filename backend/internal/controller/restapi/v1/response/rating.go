package response

import (
	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromRating(r *domain.Rating) openapi.RatingResponse {
	if r == nil {
		return openapi.RatingResponse{}
	}
	res := openapi.RatingResponse{
		ID:          httputil.Ptr(r.ID.String()),
		ChallengeID: httputil.Ptr(r.ChallengeID.String()),
		UserID:      httputil.Ptr(r.UserID.String()),
		TeamID:      httputil.Ptr(r.TeamID.String()),
		Value:       httputil.Ptr(r.Value),
		Review:      httputil.Ptr(r.Review),
		CreatedAt:   httputil.Ptr(r.CreatedAt),
	}
	return res
}

func FromRatingList(items []*domain.Rating) []openapi.RatingResponse {
	return lo.Map(items, func(item *domain.Rating, _ int) openapi.RatingResponse { return FromRating(item) })
}
