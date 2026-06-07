package apperr

import "errors"

var (
	ErrTopicNotFound     = errors.New("topic not found")
	ErrTopicNameRequired = errors.New("name is required")
)
