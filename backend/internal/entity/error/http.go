package entityError

type HTTPError struct {
	Err        error
	StatusCode int
	Code       string
}

func (e *HTTPError) Error() string   { return e.Err.Error() }
func (e *HTTPError) Unwrap() error   { return e.Err }
func (e *HTTPError) HTTPStatus() int { return e.StatusCode }
