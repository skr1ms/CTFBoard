package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrChallengeNotFound = &HTTPError{
		Err:        errors.New("challenge not found"),
		StatusCode: http.StatusNotFound,
		Code:       "CHALLENGE_NOT_FOUND",
	}
	ErrUserMustBeInTeam = &HTTPError{
		Err:        errors.New("user must be in a team to submit flags"),
		StatusCode: http.StatusForbidden,
		Code:       "USER_NOT_IN_TEAM",
	}
	ErrInvalidFlagFormat = &HTTPError{
		Err:        errors.New("invalid flag format"),
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_FLAG_FORMAT",
	}
)
