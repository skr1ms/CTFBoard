package apperr

import "errors"

var (
	ErrTokenRequired    = errors.New("token is required")
	ErrTokenNotFound    = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token expired")
	ErrTokenAlreadyUsed = errors.New("token already used")
)
