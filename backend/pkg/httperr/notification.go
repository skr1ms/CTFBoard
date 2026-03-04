package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrNotificationNotFound = &HTTPError{
		Err:        errors.New("notification not found"),
		StatusCode: http.StatusNotFound,
		Code:       "NOTIFICATION_NOT_FOUND",
	}
	ErrNotificationTitleContentRequired = &HTTPError{
		Err:        errors.New("title and content are required"),
		StatusCode: http.StatusBadRequest,
		Code:       "NOTIFICATION_TITLE_CONTENT_REQUIRED",
	}
)
