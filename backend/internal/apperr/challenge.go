package apperr

import "errors"

var (
	ErrChallengeNotFound                      = errors.New("challenge not found")
	ErrInvalidFlagFormat                      = errors.New("invalid flag format")
	ErrInvalidScoringRange                    = errors.New("initialValue must be greater than or equal to minValue for dynamic scoring")
	ErrChallengeFlagRequiredWhenSwitchingMode = errors.New("flag is required when switching to or from regex mode")
	ErrRequirementsNotMet                     = errors.New("requirements not met")
	ErrMaxAttemptsReached                     = errors.New("max attempts reached for this challenge")
	ErrChallengeLocked                        = errors.New("submissions are disabled for this challenge")
)
