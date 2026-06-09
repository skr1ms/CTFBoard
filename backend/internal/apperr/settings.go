package apperr

import "errors"

var (
	ErrAppSettingsNotFound                   = errors.New("app settings not found")
	ErrSettingsCannotChangeDuringCompetition = errors.New("cannot change registration_open while competition is active, frozen, or paused")
	ErrSettingsConflict                      = errors.New("settings were modified by another user, please retry")
)
