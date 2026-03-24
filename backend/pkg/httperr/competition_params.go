package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrCompetitionParamNotFound         = New(errors.New("competition param not found"), http.StatusNotFound, "COMPETITION_PARAM_NOT_FOUND")
	ErrCompetitionParamKeyRequired      = New(errors.New("competition param key is required"), http.StatusBadRequest, "COMPETITION_PARAM_KEY_REQUIRED")
	ErrCompetitionParamInvalidValueType = New(errors.New("invalid value type or value does not match type"), http.StatusBadRequest, "COMPETITION_PARAM_INVALID_VALUE_TYPE")
)
