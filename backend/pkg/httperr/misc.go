package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrDebugNotEnabled                   = New(errors.New("debug endpoint is not enabled"), http.StatusNotFound, "NOT_FOUND")
	ErrScoreboardHidden                  = New(errors.New("scoreboard is currently hidden"), http.StatusForbidden, "SCOREBOARD_HIDDEN")
	ErrScoreboardAdminsOnly              = New(errors.New("scoreboard is only available to administrators"), http.StatusForbidden, "SCOREBOARD_ADMINS_ONLY")
	ErrScoreboardAccessDenied            = New(errors.New("scoreboard access denied"), http.StatusForbidden, "SCOREBOARD_ACCESS_DENIED")
	ErrWebsocketOriginNotConfigured      = New(errors.New("websocket origin not configured"), http.StatusForbidden, "WEBSOCKET_ORIGIN_NOT_CONFIGURED")
	ErrWebsocketWildcardOriginNotAllowed = New(errors.New("ALLOWED_ORIGINS=* is not allowed for security; set explicit origins"), http.StatusForbidden, "WEBSOCKET_WILDCARD_ORIGIN_NOT_ALLOWED")
)
