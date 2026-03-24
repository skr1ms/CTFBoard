package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrUserNotFound                = New(errors.New("user not found"), http.StatusNotFound, "USER_NOT_FOUND")
	ErrUserAlreadyExists           = New(errors.New("user already exists"), http.StatusConflict, "USER_ALREADY_EXISTS")
	ErrUsernameTaken               = New(errors.New("username already taken"), http.StatusConflict, "USERNAME_TAKEN")
	ErrRegistrationClosed          = New(errors.New("registration is closed"), http.StatusForbidden, "REGISTRATION_CLOSED")
	ErrInvalidCredentials          = New(errors.New("invalid credentials"), http.StatusUnauthorized, "INVALID_CREDENTIALS")
	ErrAuthorizationHeaderRequired = New(errors.New("authorization header required"), http.StatusUnauthorized, "AUTHORIZATION_REQUIRED")
	ErrInvalidAuthorizationHeader  = New(errors.New("invalid authorization header format"), http.StatusUnauthorized, "INVALID_AUTHORIZATION_HEADER")
	ErrInvalidToken                = New(errors.New("invalid token"), http.StatusUnauthorized, "INVALID_TOKEN")
	ErrAccessDenied                = New(errors.New("access denied"), http.StatusForbidden, "ACCESS_DENIED")
	ErrUserBanned                  = New(errors.New("user is banned"), http.StatusForbidden, "USER_BANNED")
	ErrCaptainCannotBeDeleted      = New(errors.New("transfer captain first, then delete user"), http.StatusConflict, "CAPTAIN_CANNOT_BE_DELETED")
)
