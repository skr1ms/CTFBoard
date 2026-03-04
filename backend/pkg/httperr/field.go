package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrFieldNotFound = &HTTPError{
		Err:        errors.New("field not found"),
		StatusCode: http.StatusNotFound,
		Code:       "FIELD_NOT_FOUND",
	}
	ErrFieldUnknown = &HTTPError{
		Err:        errors.New("unknown field"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_UNKNOWN",
	}
	ErrFieldRequired = &HTTPError{
		Err:        errors.New("field is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_REQUIRED",
	}
	ErrFieldInvalidNumber = &HTTPError{
		Err:        errors.New("must be a number"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_INVALID_NUMBER",
	}
	ErrFieldInvalidBoolean = &HTTPError{
		Err:        errors.New("must be true or false"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_INVALID_BOOLEAN",
	}
	ErrFieldInvalidOption = &HTTPError{
		Err:        errors.New("invalid option"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_INVALID_OPTION",
	}
	ErrFieldTextTooLong = &HTTPError{
		Err:        errors.New("text too long (max 500)"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_TEXT_TOO_LONG",
	}
)
