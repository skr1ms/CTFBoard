package apperr

import "errors"

var (
	ErrTagNotFound     = errors.New("tag not found")
	ErrTagNameRequired = errors.New("name is required")
)
