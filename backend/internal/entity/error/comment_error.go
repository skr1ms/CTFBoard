package entityError

import (
	"errors"
	"net/http"
)

var ErrCommentNotFound = &HTTPError{
	Err:        errors.New("comment not found"),
	StatusCode: http.StatusNotFound,
	Code:       "COMMENT_NOT_FOUND",
}

var ErrCommentForbidden = &HTTPError{
	Err:        errors.New("not allowed to modify this comment"),
	StatusCode: http.StatusForbidden,
	Code:       "COMMENT_FORBIDDEN",
}
