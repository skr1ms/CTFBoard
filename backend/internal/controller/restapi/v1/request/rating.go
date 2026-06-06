package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func PutChallengeRatingRequestToParams(req *openapi.PutChallengeRatingRequest) (value int, review string, err error) {
	value = req.Value
	if req.Review != nil {
		review = *req.Review
	}

	return value, review, nil
}
