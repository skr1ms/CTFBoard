package apperr

import "errors"

var (
	ErrSolveNotFound        = errors.New("solve not found")
	ErrAlreadySolved        = errors.New("already solved")
	ErrSolutionAccessDenied = errors.New("solution access denied: challenge not solved")
)
