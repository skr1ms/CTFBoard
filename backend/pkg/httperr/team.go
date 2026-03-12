package httperr

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
	ErrMaxTeamsReached = &HTTPError{
		Err:        errors.New("maximum number of teams reached"),
		StatusCode: http.StatusConflict,
		Code:       "MAX_TEAMS_REACHED",
	}
	ErrNotCaptain = &HTTPError{
		Err:        errors.New("only captain can perform this action"),
		StatusCode: http.StatusForbidden,
		Code:       "NOT_CAPTAIN",
	}
	ErrCaptainCannotLeave = &HTTPError{
		Err:        errors.New("captain must transfer captainship before leaving the team"),
		StatusCode: http.StatusForbidden,
		Code:       "CAPTAIN_CANNOT_LEAVE",
	}
	ErrCannotLeaveAsOnlyMember = &HTTPError{
		Err:        errors.New("cannot leave team as only member, delete team instead"),
		StatusCode: http.StatusConflict,
		Code:       "CANNOT_LEAVE_AS_ONLY_MEMBER",
	}
	ErrTeamBelowMinSize = &HTTPError{
		Err:        errors.New("team does not meet minimum size requirement"),
		StatusCode: http.StatusConflict,
		Code:       "TEAM_BELOW_MIN_SIZE",
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
	ErrSoloModeNotAllowed = &HTTPError{
		Err:        errors.New("solo mode is not allowed for this competition"),
		StatusCode: http.StatusForbidden,
		Code:       "SOLO_MODE_NOT_ALLOWED",
	}
	ErrTeamsNotAllowed = &HTTPError{
		Err:        errors.New("team mode is not allowed for this competition"),
		StatusCode: http.StatusForbidden,
		Code:       "TEAMS_NOT_ALLOWED",
	}
	ErrTeamModeRequired = &HTTPError{
		Err:        errors.New("this competition requires team participation"),
		StatusCode: http.StatusForbidden,
		Code:       "TEAM_MODE_REQUIRED",
	}
	ErrSoloModeRequired = &HTTPError{
		Err:        errors.New("this competition requires solo participation"),
		StatusCode: http.StatusForbidden,
		Code:       "SOLO_MODE_REQUIRED",
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
	ErrUserWasInBannedTeam = &HTTPError{
		Err:        errors.New("user was member of a banned team and cannot create or join a team"),
		StatusCode: http.StatusForbidden,
		Code:       "USER_WAS_IN_BANNED_TEAM",
	}
	ErrInviteExpired = &HTTPError{
		Err:        errors.New("invite token has expired"),
		StatusCode: http.StatusGone,
		Code:       "INVITE_EXPIRED",
	}
	ErrTeamConflict = &HTTPError{
		Err:        errors.New("team conflict"),
		StatusCode: http.StatusConflict,
		Code:       "TEAM_CONFLICT",
	}
	ErrCannotAddToSoloTeam = &HTTPError{
		Err:        errors.New("cannot add members to a solo team"),
		StatusCode: http.StatusBadRequest,
		Code:       "CANNOT_ADD_TO_SOLO_TEAM",
	}
	ErrTeamMemberNotFound = &HTTPError{
		Err:        errors.New("team member not found"),
		StatusCode: http.StatusNotFound,
		Code:       "TEAM_MEMBER_NOT_FOUND",
	}
	ErrCannotKickSelf = &HTTPError{
		Err:        errors.New("cannot kick yourself from the team"),
		StatusCode: http.StatusBadRequest,
		Code:       "CANNOT_KICK_SELF",
	}
	ErrCannotKickCaptain = &HTTPError{
		Err:        errors.New("cannot kick the team captain"),
		StatusCode: http.StatusForbidden,
		Code:       "CANNOT_KICK_CAPTAIN",
	}
	ErrUserNotInTeam = &HTTPError{
		Err:        errors.New("user must be in a team"),
		StatusCode: http.StatusNotFound,
		Code:       "USER_NOT_IN_TEAM",
	}
	ErrCannotLeaveSoloTeam = &HTTPError{
		Err:        errors.New("cannot leave solo team in solo-only competition"),
		StatusCode: http.StatusForbidden,
		Code:       "CANNOT_LEAVE_SOLO_TEAM",
	}
	ErrCannotDisbandSoloTeam = &HTTPError{
		Err:        errors.New("cannot disband solo team in solo-only competition"),
		StatusCode: http.StatusForbidden,
		Code:       "CANNOT_DISBAND_SOLO_TEAM",
	}
)
