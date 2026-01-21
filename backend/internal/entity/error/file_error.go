package entityError

import "errors"

var (
	ErrFileNotFound = errors.New("file not found")
)
