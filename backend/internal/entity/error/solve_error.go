package entityError

import "errors"

var (
	ErrSolveNotFound = errors.New("solve not found")
	ErrAlreadySolved = errors.New("already solved")
)
