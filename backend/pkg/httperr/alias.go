package httperr

import (
	kit "github.com/wahrwelt-kit/go-httpkit/httperr"
)

type HTTPError = kit.HTTPError

var (
	New                         = kit.New
	NewValidationErrorf         = kit.NewValidationErrorf
	ErrNotAuthenticated         = kit.ErrNotAuthenticated
	ErrNotAuthenticatedSentinel = kit.ErrNotAuthenticatedSentinel
	ErrTooManyRequests          = kit.ErrTooManyRequests
)
