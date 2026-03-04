package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrTagNotFound = &HTTPError{
		Err:        errors.New("tag not found"),
		StatusCode: http.StatusNotFound,
		Code:       "TAG_NOT_FOUND",
	}
	ErrTagNameRequired = &HTTPError{
		Err:        errors.New("name is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "TAG_NAME_REQUIRED",
	}
)
