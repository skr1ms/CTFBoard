package entityError

import "errors"

var (
	ErrTeamNotFound      = errors.New("team not found")
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrUserAlreadyInTeam = errors.New("user already in team")
)
