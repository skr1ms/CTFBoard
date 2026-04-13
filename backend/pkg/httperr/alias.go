package httperr

import (
	kit "github.com/wahrwelt-kit/go-httpkit/httperr"
)

// HTTPError is an alias for the HTTP error type from go-httpkit, carrying an HTTP status code and message.
type HTTPError = kit.HTTPError

var (
	// New constructs an HTTPError with the given HTTP status code and message.
	New = kit.New
)
