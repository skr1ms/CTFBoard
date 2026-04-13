package apperr

import "errors"

var (
	ErrFieldNotFound       = errors.New("field not found")
	ErrFieldInvalidNumber  = errors.New("must be a number")
	ErrFieldInvalidBoolean = errors.New("must be true or false")
	ErrFieldTextTooLong    = errors.New("text too long (max 500)")
)
