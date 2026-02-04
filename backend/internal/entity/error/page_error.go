package entityError

import (
	"errors"
	"net/http"
)

var ErrPageNotFound = &HTTPError{
	Err:        errors.New("page not found"),
	StatusCode: http.StatusNotFound,
	Code:       "PAGE_NOT_FOUND",
}

var ErrPageSlugConflict = &HTTPError{
	Err:        errors.New("page slug already exists"),
	StatusCode: http.StatusConflict,
	Code:       "PAGE_SLUG_CONFLICT",
}
