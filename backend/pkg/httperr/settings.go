package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrAppSettingsNotFound = &HTTPError{
		Err:        errors.New("app settings not found"),
		StatusCode: http.StatusNotFound,
		Code:       "APP_SETTINGS_NOT_FOUND",
	}
	ErrSettingsCannotChangeDuringCompetition = &HTTPError{
		Err:        errors.New("cannot change scoreboard_visible or registration_open while competition is active, frozen, or paused"),
		StatusCode: http.StatusForbidden,
		Code:       "SETTINGS_CANNOT_CHANGE_DURING_COMPETITION",
	}
	ErrSettingsConflict = &HTTPError{
		Err:        errors.New("settings were modified by another user, please retry"),
		StatusCode: http.StatusConflict,
		Code:       "SETTINGS_CONFLICT",
	}
)
