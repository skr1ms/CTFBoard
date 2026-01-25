package entityError

import (
	"errors"
	"net/http"
)

var (
	ErrSolveNotFound = &HTTPError{
		Err:        errors.New("solve not found"),
		StatusCode: http.StatusNotFound,
		Code:       "SOLVE_NOT_FOUND",
	}
	ErrAlreadySolved = &HTTPError{
		Err:        errors.New("already solved"),
		StatusCode: http.StatusConflict,
		Code:       "ALREADY_SOLVED",
	}
)
