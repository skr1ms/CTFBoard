package httperr

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
		Err:        errors.New("user must be in a team"),
		StatusCode: http.StatusForbidden,
		Code:       "USER_NOT_IN_TEAM",
	}
	ErrInvalidFlagFormat = &HTTPError{
		Err:        errors.New("invalid flag format"),
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_FLAG_FORMAT",
	}
	ErrInvalidScoringRange = &HTTPError{
		Err:        errors.New("initialValue must be greater than or equal to minValue for dynamic scoring"),
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_SCORING_RANGE",
	}
	ErrChallengeFlagRequiredWhenSwitchingMode = &HTTPError{
		Err:        errors.New("flag is required when switching to or from regex mode"),
		StatusCode: http.StatusBadRequest,
		Code:       "CHALLENGE_FLAG_REQUIRED",
	}
	ErrRequirementsNotMet = &HTTPError{
		Err:        errors.New("requirements not met"),
		StatusCode: http.StatusForbidden,
		Code:       "REQUIREMENTS_NOT_MET",
	}
)
