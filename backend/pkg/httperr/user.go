package httperr

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
	ErrUsernameTaken = &HTTPError{
		Err:        errors.New("username already taken"),
		StatusCode: http.StatusConflict,
		Code:       "USERNAME_TAKEN",
	}
	ErrRegistrationClosed = &HTTPError{
		Err:        errors.New("registration is closed"),
		StatusCode: http.StatusForbidden,
		Code:       "REGISTRATION_CLOSED",
	}
	ErrInvalidCredentials = &HTTPError{
		Err:        errors.New("invalid credentials"),
		StatusCode: http.StatusUnauthorized,
		Code:       "INVALID_CREDENTIALS",
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
	ErrAuthorizationHeaderRequired = &HTTPError{
		Err:        errors.New("authorization header required"),
		StatusCode: http.StatusUnauthorized,
		Code:       "AUTHORIZATION_REQUIRED",
	}
	ErrInvalidAuthorizationHeader = &HTTPError{
		Err:        errors.New("invalid authorization header format"),
		StatusCode: http.StatusUnauthorized,
		Code:       "INVALID_AUTHORIZATION_HEADER",
	}
	ErrInvalidToken = &HTTPError{
		Err:        errors.New("invalid token"),
		StatusCode: http.StatusUnauthorized,
		Code:       "INVALID_TOKEN",
	}
	ErrTooManyRequests = &HTTPError{
		Err:        errors.New("too many requests"),
		StatusCode: http.StatusTooManyRequests,
		Code:       "RATE_LIMIT_EXCEEDED",
	}
	ErrAccessDenied = &HTTPError{
		Err:        errors.New("access denied"),
		StatusCode: http.StatusForbidden,
		Code:       "ACCESS_DENIED",
	}
	ErrUserBanned = &HTTPError{
		Err:        errors.New("user is banned"),
		StatusCode: http.StatusForbidden,
		Code:       "USER_BANNED",
	}
)
