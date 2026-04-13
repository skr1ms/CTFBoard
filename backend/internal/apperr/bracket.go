package apperr

import "errors"

var (
	ErrBracketNotFound     = errors.New("bracket not found")
	ErrBracketNameConflict = errors.New("bracket name already exists")
	ErrBracketNameRequired = errors.New("name is required")
)
