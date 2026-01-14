package entityError

import "errors"

var (
	ErrChallengeNotFound = errors.New("challenge not found")
	ErrUserMustBeInTeam  = errors.New("user must be in a team to submit flags")
)
