package entityError

import (
	"errors"
	"net/http"
)

var ErrCTFEventNotFound = &HTTPError{
	Err:        errors.New("ctf event not found"),
	StatusCode: http.StatusNotFound,
	Code:       "CTF_EVENT_NOT_FOUND",
}

var ErrGlobalRatingNotFound = &HTTPError{
	Err:        errors.New("global rating not found"),
	StatusCode: http.StatusNotFound,
	Code:       "GLOBAL_RATING_NOT_FOUND",
}
