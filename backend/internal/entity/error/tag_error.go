package entityError

import (
	"errors"
	"net/http"
)

var ErrTagNotFound = &HTTPError{
	Err:        errors.New("tag not found"),
	StatusCode: http.StatusNotFound,
	Code:       "TAG_NOT_FOUND",
}
