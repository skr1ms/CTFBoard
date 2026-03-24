package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrCompetitionNotFound           = New(errors.New("competition not found"), http.StatusNotFound, "COMPETITION_NOT_FOUND")
	ErrCompetitionNotStarted         = New(errors.New("competition has not started yet"), http.StatusForbidden, "COMPETITION_NOT_STARTED")
	ErrCompetitionEnded              = New(errors.New("competition has ended"), http.StatusForbidden, "COMPETITION_ENDED")
	ErrCompetitionPaused             = New(errors.New("competition is paused"), http.StatusForbidden, "COMPETITION_PAUSED")
	ErrSubmissionNotAllowed          = New(errors.New("submissions are not allowed at this time"), http.StatusForbidden, "SUBMISSION_NOT_ALLOWED")
	ErrCommentsAvailableAfterEnd     = New(errors.New("comments available only after competition has ended"), http.StatusForbidden, "COMPETITION_NOT_ENDED")
	ErrCompetitionActiveCannotUpdate = New(errors.New("cannot update competition mode, times, or team size while competition is active, frozen, or paused"), http.StatusForbidden, "COMPETITION_ACTIVE_CANNOT_UPDATE")
)
