package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrSolveNotFound = &HTTPError{
		Err:        errors.New("solve not found"),
		StatusCode: http.StatusNotFound,
		Code:       "SOLVE_NOT_FOUND",
	}
	ErrAlreadySolved = &HTTPError{
		Err:        errors.New("already solved"),
		StatusCode: http.StatusConflict,
		Code:       "ALREADY_SOLVED",
	}
	ErrSolutionAccessDenied = &HTTPError{
		Err:        errors.New("solution access denied: challenge not solved"),
		StatusCode: http.StatusForbidden,
		Code:       "FORBIDDEN",
	}
)

// IsExpectedClientError returns true for errors that represent normal user flow
// (e.g. resubmit, duplicate, not found) and should be logged as Info, not Error.
func IsExpectedClientError(err error) bool {
	return errors.Is(err, ErrAlreadySolved) || errors.Is(err, ErrHintAlreadyUnlocked) ||
		errors.Is(err, ErrNoTeamSelected) || errors.Is(err, ErrUserAlreadyInTeam) ||
		errors.Is(err, ErrUserAlreadyExists) || errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrTeamFull) ||
		errors.Is(err, ErrPageSlugConflict) || errors.Is(err, ErrBracketNameConflict)
}
