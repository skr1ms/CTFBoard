package entityError

import "errors"

var (
	ErrCompetitionNotFound   = errors.New("competition not found")
	ErrCompetitionNotActive  = errors.New("competition is not active")
	ErrCompetitionNotStarted = errors.New("competition has not started yet")
	ErrCompetitionEnded      = errors.New("competition has ended")
	ErrCompetitionPaused     = errors.New("competition is paused")
)
