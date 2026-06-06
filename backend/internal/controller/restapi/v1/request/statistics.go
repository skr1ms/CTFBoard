package request

import (
	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/openapi"
)

func SubmissionTypeIsCorrect(pType openapi.GetStatisticsSubmissionsTypeParamsType) (bool, error) {
	if pType == openapi.Correct {
		return true, nil
	}

	if pType == openapi.Incorrect {
		return false, nil
	}

	return false, apperr.NewValidationErrorf("type must be 'correct' or 'incorrect'")
}
