package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrCompetitionNotFound = &HTTPError{
		Err:        errors.New("competition not found"),
		StatusCode: http.StatusNotFound,
		Code:       "COMPETITION_NOT_FOUND",
	}
	ErrCompetitionNotActive = &HTTPError{
		Err:        errors.New("competition is not active"),
		StatusCode: http.StatusForbidden,
		Code:       "COMPETITION_NOT_ACTIVE",
	}
	ErrCompetitionNotStarted = &HTTPError{
		Err:        errors.New("competition has not started yet"),
		StatusCode: http.StatusForbidden,
		Code:       "COMPETITION_NOT_STARTED",
	}
	ErrCompetitionEnded = &HTTPError{
		Err:        errors.New("competition has ended"),
		StatusCode: http.StatusForbidden,
		Code:       "COMPETITION_ENDED",
	}
	ErrCompetitionPaused = &HTTPError{
		Err:        errors.New("competition is paused"),
		StatusCode: http.StatusForbidden,
		Code:       "COMPETITION_PAUSED",
	}
)
