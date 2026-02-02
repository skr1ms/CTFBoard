package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrTeamNotFound = &HTTPError{
		Err:        errors.New("team not found"),
		StatusCode: http.StatusNotFound,
		Code:       "TEAM_NOT_FOUND",
	}
	ErrTeamAlreadyExists = &HTTPError{
		Err:        errors.New("team already exists"),
		StatusCode: http.StatusConflict,
		Code:       "TEAM_ALREADY_EXISTS",
	}
	ErrUserAlreadyInTeam = &HTTPError{
		Err:        errors.New("user already in team"),
		StatusCode: http.StatusConflict,
		Code:       "USER_ALREADY_IN_TEAM",
	}
	ErrTeamFull = &HTTPError{
		Err:        errors.New("team is full"),
		StatusCode: http.StatusConflict,
		Code:       "TEAM_FULL",
	}
	ErrNotCaptain = &HTTPError{
		Err:        errors.New("only captain can perform this action"),
		StatusCode: http.StatusForbidden,
		Code:       "NOT_CAPTAIN",
	}
	ErrCannotLeaveAsOnlyMember = &HTTPError{
		Err:        errors.New("cannot leave team as only member, delete team instead"),
		StatusCode: http.StatusConflict,
		Code:       "CANNOT_LEAVE_AS_ONLY_MEMBER",
	}
	ErrNewCaptainNotInTeam = &HTTPError{
		Err:        errors.New("new captain must be a member of the team"),
		StatusCode: http.StatusBadRequest,
		Code:       "NEW_CAPTAIN_NOT_IN_TEAM",
	}
	ErrCannotTransferToSelf = &HTTPError{
		Err:        errors.New("cannot transfer captainship to yourself"),
		StatusCode: http.StatusBadRequest,
		Code:       "CANNOT_TRANSFER_TO_SELF",
	}
	ErrNoTeamSelected = &HTTPError{
		Err:        errors.New("user has not selected a participation mode"),
		StatusCode: http.StatusBadRequest,
		Code:       "NO_TEAM_SELECTED",
	}
	ErrCannotSwitchTeams = &HTTPError{
		Err:        errors.New("team switching is disabled for this competition"),
		StatusCode: http.StatusForbidden,
		Code:       "CANNOT_SWITCH_TEAMS",
	}
	ErrInvalidTransition = &HTTPError{
		Err:        errors.New("invalid participation state transition"),
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_TRANSITION",
	}
	ErrSoloModeNotAllowed = &HTTPError{
		Err:        errors.New("solo mode is not allowed for this competition"),
		StatusCode: http.StatusForbidden,
		Code:       "SOLO_MODE_NOT_ALLOWED",
	}
	ErrTeamModeRequired = &HTTPError{
		Err:        errors.New("this competition requires team participation"),
		StatusCode: http.StatusForbidden,
		Code:       "TEAM_MODE_REQUIRED",
	}
	ErrConfirmationRequired = &HTTPError{
		Err:        errors.New("confirmation required for this action"),
		StatusCode: http.StatusBadRequest,
		Code:       "CONFIRMATION_REQUIRED",
	}
	ErrRosterFrozen = &HTTPError{
		Err:        errors.New("team roster is frozen"),
		StatusCode: http.StatusForbidden,
		Code:       "ROSTER_FROZEN",
	}
	ErrEmailNotVerified = &HTTPError{
		Err:        errors.New("email verification required"),
		StatusCode: http.StatusForbidden,
		Code:       "EMAIL_NOT_VERIFIED",
	}
	ErrTeamBanned = &HTTPError{
		Err:        errors.New("team is banned"),
		StatusCode: http.StatusForbidden,
		Code:       "TEAM_BANNED",
	}
)
