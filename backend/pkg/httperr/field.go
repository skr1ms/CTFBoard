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

	ErrFieldTextTooLong = &HTTPError{
		Err:        errors.New("text too long (max 500)"),
		StatusCode: http.StatusBadRequest,
		Code:       "FIELD_TEXT_TOO_LONG",
	}
)
