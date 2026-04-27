package apperr

import "errors"

var (
	ErrDebugNotEnabled                   = errors.New("debug endpoint is not enabled")
	ErrScoreboardHidden                  = errors.New("scoreboard is currently hidden")
	ErrScoreboardAdminsOnly              = errors.New("scoreboard is only available to administrators")
	ErrScoreboardAccessDenied            = errors.New("scoreboard access denied")
	ErrWebsocketOriginNotConfigured      = errors.New("websocket origin not configured")
	ErrWebsocketWildcardOriginNotAllowed = errors.New("wildcard ALLOWED_ORIGINS=* is not permitted; set explicit origins")
	ErrVisibilityForbidden               = errors.New("not found")
	ErrSetupAlreadyComplete              = errors.New("setup has already been completed")
	ErrSetupRequired                     = errors.New("setup wizard must be completed before using this platform")
)
