package entityError

import (
	"errors"
	"net/http"
)

var ErrBracketNotFound = &HTTPError{
	Err:        errors.New("bracket not found"),
	StatusCode: http.StatusNotFound,
	Code:       "BRACKET_NOT_FOUND",
}

var ErrBracketNameConflict = &HTTPError{
	Err:        errors.New("bracket name already exists"),
	StatusCode: http.StatusConflict,
	Code:       "BRACKET_NAME_CONFLICT",
}
