package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrBracketNotFound = &HTTPError{
		Err:        errors.New("bracket not found"),
		StatusCode: http.StatusNotFound,
		Code:       "BRACKET_NOT_FOUND",
	}
	ErrBracketNameConflict = &HTTPError{
		Err:        errors.New("bracket name already exists"),
		StatusCode: http.StatusConflict,
		Code:       "BRACKET_NAME_CONFLICT",
	}
	ErrBracketNameRequired = &HTTPError{
		Err:        errors.New("name is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "BRACKET_NAME_REQUIRED",
	}
)
