package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrBracketNotFound     = New(errors.New("bracket not found"), http.StatusNotFound, "BRACKET_NOT_FOUND")
	ErrBracketNameConflict = New(errors.New("bracket name already exists"), http.StatusConflict, "BRACKET_NAME_CONFLICT")
	ErrBracketNameRequired = New(errors.New("name is required"), http.StatusBadRequest, "BRACKET_NAME_REQUIRED")
)
