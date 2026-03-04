package httperr

import (
	"errors"
	"net/http"
)

var ErrSubmissionNotFound = &HTTPError{
	Err:        errors.New("submission not found"),
	StatusCode: http.StatusNotFound,
	Code:       "SUBMISSION_NOT_FOUND",
}
