package entityError

import (
	"errors"
	"net/http"
)

var ErrFieldNotFound = &HTTPError{
	Err:        errors.New("field not found"),
	StatusCode: http.StatusNotFound,
	Code:       "FIELD_NOT_FOUND",
}
