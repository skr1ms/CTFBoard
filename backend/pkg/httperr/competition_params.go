package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrCompetitionParamNotFound = &HTTPError{
		Err:        errors.New("competition param not found"),
		StatusCode: http.StatusNotFound,
		Code:       "COMPETITION_PARAM_NOT_FOUND",
	}
	ErrCompetitionParamKeyRequired = &HTTPError{
		Err:        errors.New("competition param key is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "COMPETITION_PARAM_KEY_REQUIRED",
	}
	ErrCompetitionParamInvalidValueType = &HTTPError{
		Err:        errors.New("invalid value type or value does not match type"),
		StatusCode: http.StatusBadRequest,
		Code:       "COMPETITION_PARAM_INVALID_VALUE_TYPE",
	}
)
