package helper

import (
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func New(err error, status int, code string) *httperr.HTTPError {
	return httperr.New(err, status, code)
}

func NewValidationErrorf(format string, args ...any) *httperr.HTTPError {
	return httperr.NewValidationErrorf(format, args...)
}

func IsExpectedClientError(err error) bool {
	return httperr.IsExpectedClientError(err)
}

var (
	ErrDebugNotEnabled       = httperr.ErrDebugNotEnabled
	ErrTooManyRequests       = httperr.ErrTooManyRequests
	ErrCompetitionNotStarted = httperr.ErrCompetitionNotStarted
	ErrTeamBanned            = httperr.ErrTeamBanned
	ErrUserBanned            = httperr.ErrUserBanned
	ErrTokenRequired         = httperr.ErrTokenRequired
	ErrNotAuthenticated      = httperr.ErrNotAuthenticated
	ErrAccessDenied          = httperr.ErrAccessDenied
	ErrUserMustBeInTeam      = httperr.ErrUserMustBeInTeam
	ErrUserNotInTeam         = httperr.ErrUserNotInTeam

	ErrWriteupsDisabled   = httperr.ErrWriteupsDisabled
	ErrOAuthStateMissing  = httperr.ErrOAuthStateMissing
	ErrOAuthStateMismatch = httperr.ErrOAuthStateMismatch

	ErrAlreadySolved = httperr.ErrAlreadySolved
)
