package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrUserNotFound = &HTTPError{
		Err:        errors.New("user not found"),
		StatusCode: http.StatusNotFound,
		Code:       "USER_NOT_FOUND",
	}
	ErrUserAlreadyExists = &HTTPError{
		Err:        errors.New("user already exists"),
		StatusCode: http.StatusConflict,
		Code:       "USER_ALREADY_EXISTS",
	}
	ErrInvalidCredentials = &HTTPError{
		Err:        errors.New("invalid credentials"),
		StatusCode: http.StatusUnauthorized,
		Code:       "INVALID_CREDENTIALS",
	}
	ErrTokenNotFound = &HTTPError{
		Err:        errors.New("token not found"),
		StatusCode: http.StatusNotFound,
		Code:       "TOKEN_NOT_FOUND",
	}
	ErrTokenExpired = &HTTPError{
		Err:        errors.New("token expired"),
		StatusCode: http.StatusBadRequest,
		Code:       "TOKEN_EXPIRED",
	}
	ErrTokenAlreadyUsed = &HTTPError{
		Err:        errors.New("token already used"),
		StatusCode: http.StatusBadRequest,
		Code:       "TOKEN_ALREADY_USED",
	}
	ErrUserNotVerified = &HTTPError{
		Err:        errors.New("email not verified"),
		StatusCode: http.StatusUnauthorized,
		Code:       "USER_NOT_VERIFIED",
	}
	ErrNotAuthenticated = &HTTPError{
		Err:        errors.New("not authenticated"),
		StatusCode: http.StatusUnauthorized,
		Code:       "NOT_AUTHENTICATED",
	}
)
