package apperr

import "errors"

var (
	ErrCommentNotFound        = errors.New("comment not found")
	ErrCommentForbidden       = errors.New("not allowed to modify this comment")
	ErrCommentContentRequired = errors.New("content is required")
)
