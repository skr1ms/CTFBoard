package httperr

import (
	"errors"
	"net/http"
)

var ErrAPITokenNotFound = &HTTPError{
	Err:        errors.New("api token not found"),
	StatusCode: http.StatusNotFound,
	Code:       "API_TOKEN_NOT_FOUND",
}
