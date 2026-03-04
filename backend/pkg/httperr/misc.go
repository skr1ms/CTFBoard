package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrDebugNotEnabled = &HTTPError{
		Err:        errors.New("debug endpoint is not enabled"),
		StatusCode: http.StatusNotFound,
		Code:       "NOT_FOUND",
	}
	ErrScoreboardHidden = &HTTPError{
		Err:        errors.New("scoreboard is currently hidden"),
		StatusCode: http.StatusForbidden,
		Code:       "SCOREBOARD_HIDDEN",
	}
	ErrScoreboardAdminsOnly = &HTTPError{
		Err:        errors.New("scoreboard is only available to administrators"),
		StatusCode: http.StatusForbidden,
		Code:       "SCOREBOARD_ADMINS_ONLY",
	}
	ErrScoreboardAccessDenied = &HTTPError{
		Err:        errors.New("scoreboard access denied"),
		StatusCode: http.StatusForbidden,
		Code:       "SCOREBOARD_ACCESS_DENIED",
	}
	ErrWebsocketOriginNotConfigured = &HTTPError{
		Err:        errors.New("websocket origin not configured"),
		StatusCode: http.StatusForbidden,
		Code:       "WEBSOCKET_ORIGIN_NOT_CONFIGURED",
	}
	ErrInvalidID = &HTTPError{
		Err:        errors.New("invalid ID"),
		StatusCode: http.StatusBadRequest,
		Code:       "INVALID_ID",
	}
)
