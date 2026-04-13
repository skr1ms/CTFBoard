package apperr

import "errors"

var (
	ErrCompetitionNotFound           = errors.New("competition not found")
	ErrCompetitionNotStarted         = errors.New("competition has not started yet")
	ErrCompetitionEnded              = errors.New("competition has ended")
	ErrCompetitionPaused             = errors.New("competition is paused")
	ErrSubmissionNotAllowed          = errors.New("submissions are not allowed at this time")
	ErrCommentsAvailableAfterEnd     = errors.New("comments available only after competition has ended")
	ErrCompetitionActiveCannotUpdate = errors.New("cannot update competition mode, times, or team size while competition is active, frozen, or paused")
)
