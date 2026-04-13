package apperr

import "errors"

var (
	ErrRatingNotFound         = errors.New("rating not found")
	ErrSolveRequiredForRating = errors.New("challenge must be solved before rating")
)
