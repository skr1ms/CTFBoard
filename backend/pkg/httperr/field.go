package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrFieldNotFound       = New(errors.New("field not found"), http.StatusNotFound, "FIELD_NOT_FOUND")
	ErrFieldInvalidNumber  = New(errors.New("must be a number"), http.StatusBadRequest, "FIELD_INVALID_NUMBER")
	ErrFieldInvalidBoolean = New(errors.New("must be true or false"), http.StatusBadRequest, "FIELD_INVALID_BOOLEAN")
	ErrFieldTextTooLong    = New(errors.New("text too long (max 500)"), http.StatusBadRequest, "FIELD_TEXT_TOO_LONG")
)
