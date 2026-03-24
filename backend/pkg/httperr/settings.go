package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrAppSettingsNotFound                   = New(errors.New("app settings not found"), http.StatusNotFound, "APP_SETTINGS_NOT_FOUND")
	ErrSettingsCannotChangeDuringCompetition = New(errors.New("cannot change scoreboard_visible or registration_open while competition is active, frozen, or paused"), http.StatusForbidden, "SETTINGS_CANNOT_CHANGE_DURING_COMPETITION")
	ErrSettingsConflict                      = New(errors.New("settings were modified by another user, please retry"), http.StatusConflict, "SETTINGS_CONFLICT")
)
