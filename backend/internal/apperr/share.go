package apperr

import "errors"

var (
	ErrShareNotFound  = errors.New("share not found")
	ErrSharesDisabled = errors.New("share links are disabled")
)
