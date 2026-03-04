package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrAwardNotFound = &HTTPError{
		Err:        errors.New("award not found"),
		StatusCode: http.StatusNotFound,
		Code:       "AWARD_NOT_FOUND",
	}
	ErrAwardTeamIDRequired = &HTTPError{
		Err:        errors.New("team_id is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "AWARD_TEAM_ID_REQUIRED",
	}
	ErrAwardValueCannotBeZero = &HTTPError{
		Err:        errors.New("value cannot be 0"),
		StatusCode: http.StatusBadRequest,
		Code:       "AWARD_VALUE_CANNOT_BE_ZERO",
	}
)
