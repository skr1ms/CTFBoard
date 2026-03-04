package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrTokenRequired = &HTTPError{
		Err:        errors.New("token is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "TOKEN_REQUIRED",
	}
	ErrTokenNotFound = &HTTPError{
		Err:        errors.New("invalid token"),
		StatusCode: http.StatusNotFound,
		Code:       "TOKEN_NOT_FOUND",
	}
	ErrTokenExpired = &HTTPError{
		Err:        errors.New("token expired"),
		StatusCode: http.StatusGone,
		Code:       "TOKEN_EXPIRED",
	}
	ErrTokenAlreadyUsed = &HTTPError{
		Err:        errors.New("token already used"),
		StatusCode: http.StatusConflict,
		Code:       "TOKEN_ALREADY_USED",
	}
)
