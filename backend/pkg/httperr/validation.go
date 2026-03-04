package httperr

import (
	"fmt"
	"net/http"
)

// NewValidationErrorf creates an HTTPError with code VALIDATION_ERROR for dynamic validation messages.
func NewValidationErrorf(format string, args ...any) *HTTPError {
	return &HTTPError{
		Err:        fmt.Errorf(format, args...),
		StatusCode: http.StatusBadRequest,
		Code:       "VALIDATION_ERROR",
	}
}
