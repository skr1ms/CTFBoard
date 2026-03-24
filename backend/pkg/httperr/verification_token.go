package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrTokenRequired    = New(errors.New("token is required"), http.StatusBadRequest, "TOKEN_REQUIRED")
	ErrTokenNotFound    = New(errors.New("invalid token"), http.StatusNotFound, "TOKEN_NOT_FOUND")
	ErrTokenExpired     = New(errors.New("token expired"), http.StatusGone, "TOKEN_EXPIRED")
	ErrTokenAlreadyUsed = New(errors.New("token already used"), http.StatusConflict, "TOKEN_ALREADY_USED")
)
