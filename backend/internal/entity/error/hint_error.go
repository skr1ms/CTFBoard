package entityError

import "errors"

var (
	ErrHintNotFound        = errors.New("hint not found")
	ErrHintAlreadyUnlocked = errors.New("hint already unlocked")
	ErrInsufficientPoints  = errors.New("insufficient points to unlock hint")
)
