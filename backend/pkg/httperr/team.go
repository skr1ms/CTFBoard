package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrTeamNotFound            = New(errors.New("team not found"), http.StatusNotFound, "TEAM_NOT_FOUND")
	ErrTeamAlreadyExists       = New(errors.New("team already exists"), http.StatusConflict, "TEAM_ALREADY_EXISTS")
	ErrUserAlreadyInTeam       = New(errors.New("user already in team"), http.StatusConflict, "USER_ALREADY_IN_TEAM")
	ErrTeamFull                = New(errors.New("team is full"), http.StatusConflict, "TEAM_FULL")
	ErrMaxTeamsReached         = New(errors.New("maximum number of teams reached"), http.StatusConflict, "MAX_TEAMS_REACHED")
	ErrNotCaptain              = New(errors.New("only captain can perform this action"), http.StatusForbidden, "NOT_CAPTAIN")
	ErrCaptainCannotLeave      = New(errors.New("captain must transfer captainship before leaving the team"), http.StatusForbidden, "CAPTAIN_CANNOT_LEAVE")
	ErrCannotLeaveAsOnlyMember = New(errors.New("cannot leave team as only member, delete team instead"), http.StatusConflict, "CANNOT_LEAVE_AS_ONLY_MEMBER")
	ErrTeamBelowMinSize        = New(errors.New("team does not meet minimum size requirement"), http.StatusConflict, "TEAM_BELOW_MIN_SIZE")
	ErrNewCaptainNotInTeam     = New(errors.New("new captain must be a member of the team"), http.StatusBadRequest, "NEW_CAPTAIN_NOT_IN_TEAM")
	ErrCannotTransferToSelf    = New(errors.New("cannot transfer captainship to yourself"), http.StatusBadRequest, "CANNOT_TRANSFER_TO_SELF")
	ErrNoTeamSelected          = New(errors.New("user has not selected a participation mode"), http.StatusBadRequest, "NO_TEAM_SELECTED")
	ErrSoloModeNotAllowed      = New(errors.New("solo mode is not allowed for this competition"), http.StatusForbidden, "SOLO_MODE_NOT_ALLOWED")
	ErrTeamsNotAllowed         = New(errors.New("team mode is not allowed for this competition"), http.StatusForbidden, "TEAMS_NOT_ALLOWED")
	ErrTeamModeRequired        = New(errors.New("this competition requires team participation"), http.StatusForbidden, "TEAM_MODE_REQUIRED")
	ErrSoloModeRequired        = New(errors.New("this competition requires solo participation"), http.StatusForbidden, "SOLO_MODE_REQUIRED")
	ErrConfirmationRequired    = New(errors.New("confirmation required for this action"), http.StatusBadRequest, "CONFIRMATION_REQUIRED")
	ErrRosterFrozen            = New(errors.New("team roster is frozen"), http.StatusForbidden, "ROSTER_FROZEN")
	ErrEmailNotVerified        = New(errors.New("email verification required"), http.StatusForbidden, "EMAIL_NOT_VERIFIED")
	ErrTeamBanned              = New(errors.New("team is banned"), http.StatusForbidden, "TEAM_BANNED")
	ErrUserWasInBannedTeam     = New(errors.New("user was member of a banned team and cannot create or join a team"), http.StatusForbidden, "USER_WAS_IN_BANNED_TEAM")
	ErrInviteExpired           = New(errors.New("invite token has expired"), http.StatusGone, "INVITE_EXPIRED")
	ErrTeamConflict            = New(errors.New("team conflict"), http.StatusConflict, "TEAM_CONFLICT")
	ErrCannotAddToSoloTeam     = New(errors.New("cannot add members to a solo team"), http.StatusBadRequest, "CANNOT_ADD_TO_SOLO_TEAM")
	ErrTeamMemberNotFound      = New(errors.New("team member not found"), http.StatusNotFound, "TEAM_MEMBER_NOT_FOUND")
	ErrCannotKickSelf          = New(errors.New("cannot kick yourself from the team"), http.StatusBadRequest, "CANNOT_KICK_SELF")
	ErrCannotKickCaptain       = New(errors.New("cannot kick the team captain"), http.StatusForbidden, "CANNOT_KICK_CAPTAIN")
	ErrUserNotInTeam           = New(errors.New("user must be in a team"), http.StatusNotFound, "USER_NOT_IN_TEAM")
	ErrCannotLeaveSoloTeam     = New(errors.New("cannot leave solo team in solo-only competition"), http.StatusForbidden, "CANNOT_LEAVE_SOLO_TEAM")
	ErrCannotDisbandSoloTeam   = New(errors.New("cannot disband solo team in solo-only competition"), http.StatusForbidden, "CANNOT_DISBAND_SOLO_TEAM")
)
