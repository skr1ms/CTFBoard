package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrRatingNotFound         = New(errors.New("rating not found"), http.StatusNotFound, "RATING_NOT_FOUND")
	ErrSolveRequiredForRating = New(errors.New("challenge must be solved before rating"), http.StatusForbidden, "SOLVE_REQUIRED_FOR_RATING")
)
