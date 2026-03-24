package httperr

import (
	"errors"
	"net/http"
)

var ErrAPITokenNotFound = New(errors.New("api token not found"), http.StatusNotFound, "API_TOKEN_NOT_FOUND")
