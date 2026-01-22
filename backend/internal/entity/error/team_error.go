package entityError

import "errors"

var (
	ErrTeamNotFound            = errors.New("team not found")
	ErrTeamAlreadyExists       = errors.New("team already exists")
	ErrUserAlreadyInTeam       = errors.New("user already in team")
	ErrTeamFull                = errors.New("team is full")
	ErrNotCaptain              = errors.New("only captain can perform this action")
	ErrCannotLeaveAsOnlyMember = errors.New("cannot leave team as only member, delete team instead")
	ErrNewCaptainNotInTeam     = errors.New("new captain must be a member of the team")
	ErrCannotTransferToSelf    = errors.New("cannot transfer captainship to yourself")
)
