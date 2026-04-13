package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func PutChallengeRatingRequestToParams(req *openapi.PutChallengeRatingRequest) (value int, review string, err error) {
	if req.Value < 1 || req.Value > 5 {
		return 0, "", apperr.NewValidationErrorf("value must be between 1 and 5")
	}

	value = req.Value
	if req.Review != nil {
		review = *req.Review
	}

	if len(review) > 2000 {
		return 0, "", apperr.NewValidationErrorf("review must be at most 2000 characters")
	}

	return value, review, nil
}
