package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrCommentNotFound        = New(errors.New("comment not found"), http.StatusNotFound, "COMMENT_NOT_FOUND")
	ErrCommentForbidden       = New(errors.New("not allowed to modify this comment"), http.StatusForbidden, "COMMENT_FORBIDDEN")
	ErrCommentContentRequired = New(errors.New("content is required"), http.StatusBadRequest, "COMMENT_CONTENT_REQUIRED")
)
