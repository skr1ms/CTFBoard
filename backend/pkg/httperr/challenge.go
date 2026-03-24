package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrChallengeNotFound                      = New(errors.New("challenge not found"), http.StatusNotFound, "CHALLENGE_NOT_FOUND")
	ErrUserMustBeInTeam                       = New(errors.New("user must be in a team"), http.StatusNotFound, "USER_NOT_IN_TEAM")
	ErrInvalidFlagFormat                      = New(errors.New("invalid flag format"), http.StatusBadRequest, "INVALID_FLAG_FORMAT")
	ErrInvalidScoringRange                    = New(errors.New("initialValue must be greater than or equal to minValue for dynamic scoring"), http.StatusBadRequest, "INVALID_SCORING_RANGE")
	ErrChallengeFlagRequiredWhenSwitchingMode = New(errors.New("flag is required when switching to or from regex mode"), http.StatusBadRequest, "CHALLENGE_FLAG_REQUIRED")
	ErrRequirementsNotMet                     = New(errors.New("requirements not met"), http.StatusForbidden, "REQUIREMENTS_NOT_MET")
	ErrMaxAttemptsReached                     = New(errors.New("max attempts reached for this challenge"), http.StatusTooManyRequests, "MAX_ATTEMPTS_REACHED")
	ErrChallengeLocked                        = New(errors.New("submissions are disabled for this challenge"), http.StatusForbidden, "CHALLENGE_LOCKED")
)
