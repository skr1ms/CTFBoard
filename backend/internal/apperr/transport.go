package apperr

import "errors"

var (
	ErrNotAuthenticated = errors.New("not authenticated")
	ErrTooManyRequests  = errors.New("too many requests")
)
