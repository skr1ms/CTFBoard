package httperr

import (
	"errors"
	"net/http"
)

var ErrSubmissionNotFound = New(errors.New("submission not found"), http.StatusNotFound, "SUBMISSION_NOT_FOUND")
