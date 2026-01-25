package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrHintNotFound = &HTTPError{
		Err:        errors.New("hint not found"),
		StatusCode: http.StatusNotFound,
		Code:       "HINT_NOT_FOUND",
	}
	ErrHintAlreadyUnlocked = &HTTPError{
		Err:        errors.New("hint already unlocked"),
		StatusCode: http.StatusConflict,
		Code:       "HINT_ALREADY_UNLOCKED",
	}
	ErrInsufficientPoints = &HTTPError{
		Err:        errors.New("insufficient points to unlock hint"),
		StatusCode: http.StatusPaymentRequired,
		Code:       "INSUFFICIENT_POINTS",
	}
)
