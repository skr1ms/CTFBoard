package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrSolveNotFound = &HTTPError{
		Err:        errors.New("solve not found"),
		StatusCode: http.StatusNotFound,
		Code:       "SOLVE_NOT_FOUND",
	}
	ErrAlreadySolved = &HTTPError{
		Err:        errors.New("already solved"),
		StatusCode: http.StatusConflict,
		Code:       "ALREADY_SOLVED",
	}
	ErrSolutionAccessDenied = &HTTPError{
		Err:        errors.New("solution access denied: challenge not solved"),
		StatusCode: http.StatusForbidden,
		Code:       "FORBIDDEN",
	}
)

func IsExpectedClientError(err error) bool {
	var he *HTTPError
	if errors.As(err, &he) && he.Code == "VALIDATION_ERROR" {
		return true
	}
	return errors.Is(err, ErrAlreadySolved) || errors.Is(err, ErrHintAlreadyUnlocked) ||
		errors.Is(err, ErrNoTeamSelected) || errors.Is(err, ErrUserAlreadyInTeam) ||
		errors.Is(err, ErrUserAlreadyExists) || errors.Is(err, ErrUsernameTaken) || errors.Is(err, ErrTeamFull) ||
		errors.Is(err, ErrPageSlugConflict) || errors.Is(err, ErrBracketNameConflict) ||
		errors.Is(err, ErrTeamNotFound) || errors.Is(err, ErrUserNotFound) || errors.Is(err, ErrChallengeNotFound) ||
		errors.Is(err, ErrInviteExpired) || errors.Is(err, ErrRegistrationClosed) || errors.Is(err, ErrMaxTeamsReached) ||
		errors.Is(err, ErrRosterFrozen) || errors.Is(err, ErrTeamAlreadyExists) || errors.Is(err, ErrCaptainCannotLeave) ||
		errors.Is(err, ErrCannotLeaveAsOnlyMember) || errors.Is(err, ErrTeamBelowMinSize) || errors.Is(err, ErrEmailNotVerified) ||
		errors.Is(err, ErrRequirementsNotMet) || errors.Is(err, ErrHintNotFound) || errors.Is(err, ErrInsufficientPoints) ||
		errors.Is(err, ErrInvalidFlagFormat) || errors.Is(err, ErrSubmissionNotFound) || errors.Is(err, ErrCommentForbidden) ||
		errors.Is(err, ErrPageNotFound) || errors.Is(err, ErrBracketNotFound) || errors.Is(err, ErrNewCaptainNotInTeam) ||
		errors.Is(err, ErrCannotTransferToSelf) || errors.Is(err, ErrCannotKickSelf) || errors.Is(err, ErrCannotKickCaptain) ||
		errors.Is(err, ErrUserNotInTeam) || errors.Is(err, ErrTeamMemberNotFound) || errors.Is(err, ErrCannotAddToSoloTeam) ||
		errors.Is(err, ErrTeamConflict) || errors.Is(err, ErrCompetitionNotStarted) || errors.Is(err, ErrCompetitionEnded) ||
		errors.Is(err, ErrCompetitionPaused) || errors.Is(err, ErrSubmissionNotAllowed) || errors.Is(err, ErrCommentsAvailableAfterEnd) ||
		errors.Is(err, ErrSoloModeNotAllowed) || errors.Is(err, ErrTeamsNotAllowed) || errors.Is(err, ErrTeamModeRequired) ||
		errors.Is(err, ErrSoloModeRequired) || errors.Is(err, ErrConfirmationRequired) ||
		errors.Is(err, ErrUserBanned) || errors.Is(err, ErrTeamBanned) ||
		errors.Is(err, ErrSettingsConflict)
}
