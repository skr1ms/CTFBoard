package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrHintNotFound        = New(errors.New("hint not found"), http.StatusNotFound, "HINT_NOT_FOUND")
	ErrHintAlreadyUnlocked = New(errors.New("hint already unlocked"), http.StatusConflict, "HINT_ALREADY_UNLOCKED")
	ErrInsufficientPoints  = New(errors.New("insufficient points to unlock hint"), http.StatusPaymentRequired, "INSUFFICIENT_POINTS")
	ErrHintOrderRequired   = New(errors.New("hints must be unlocked in order"), http.StatusBadRequest, "HINT_ORDER_REQUIRED")
)
