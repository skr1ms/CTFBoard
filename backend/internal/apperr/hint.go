package apperr

import "errors"

var (
	ErrHintNotFound        = errors.New("hint not found")
	ErrHintAlreadyUnlocked = errors.New("hint already unlocked")
	ErrInsufficientPoints  = errors.New("insufficient points to unlock hint")
	ErrHintOrderRequired   = errors.New("hints must be unlocked in order")
)
