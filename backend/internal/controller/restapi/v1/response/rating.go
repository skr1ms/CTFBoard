package response

import (
	"github.com/samber/lo"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func FromRating(r *domain.Rating) openapi.RatingResponse {
	if r == nil {
		return openapi.RatingResponse{}
	}

	res := openapi.RatingResponse{
		ID:          new(r.ID.String()),
		ChallengeID: new(r.ChallengeID.String()),
		UserID:      new(r.UserID.String()),
		TeamID:      new(r.TeamID.String()),
		Value:       new(r.Value),
		Review:      new(r.Review),
		CreatedAt:   new(r.CreatedAt),
	}

	return res
}

func FromRatingList(items []*domain.Rating) []openapi.RatingResponse {
	return lo.Map(items, func(item *domain.Rating, _ int) openapi.RatingResponse { return FromRating(item) })
}
