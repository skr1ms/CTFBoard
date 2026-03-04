package httperr

import "errors"

// New returns an HTTPError with the given error, status code, and code.
func New(err error, status int, code string) *HTTPError {
	if err == nil {
		err = errors.New("")
	}
	return &HTTPError{Err: err, StatusCode: status, Code: code}
}

type HTTPError struct {
	Err        error
	StatusCode int
	Code       string
}

func (e *HTTPError) Error() string   { return e.Err.Error() }
func (e *HTTPError) Unwrap() error   { return e.Err }
func (e *HTTPError) HTTPStatus() int { return e.StatusCode }
