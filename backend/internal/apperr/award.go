package apperr

import "errors"

var (
	ErrAwardNotFound          = errors.New("award not found")
	ErrAwardTeamIDRequired    = errors.New("team_id is required")
	ErrAwardValueCannotBeZero = errors.New("value cannot be 0")
)
