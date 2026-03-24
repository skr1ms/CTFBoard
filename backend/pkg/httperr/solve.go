package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrSolveNotFound        = New(errors.New("solve not found"), http.StatusNotFound, "SOLVE_NOT_FOUND")
	ErrAlreadySolved        = New(errors.New("already solved"), http.StatusConflict, "ALREADY_SOLVED")
	ErrSolutionAccessDenied = New(errors.New("solution access denied: challenge not solved"), http.StatusForbidden, "FORBIDDEN")
)
