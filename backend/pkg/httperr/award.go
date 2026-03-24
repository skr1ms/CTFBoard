package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrAwardNotFound          = New(errors.New("award not found"), http.StatusNotFound, "AWARD_NOT_FOUND")
	ErrAwardTeamIDRequired    = New(errors.New("team_id is required"), http.StatusBadRequest, "AWARD_TEAM_ID_REQUIRED")
	ErrAwardValueCannotBeZero = New(errors.New("value cannot be 0"), http.StatusBadRequest, "AWARD_VALUE_CANNOT_BE_ZERO")
)
