package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrFileNotFound = &HTTPError{
		Err:        errors.New("file not found"),
		StatusCode: http.StatusNotFound,
		Code:       "FILE_NOT_FOUND",
	}
)
