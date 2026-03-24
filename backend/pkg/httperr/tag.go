package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrTagNotFound     = New(errors.New("tag not found"), http.StatusNotFound, "TAG_NOT_FOUND")
	ErrTagNameRequired = New(errors.New("name is required"), http.StatusBadRequest, "TAG_NAME_REQUIRED")
)
