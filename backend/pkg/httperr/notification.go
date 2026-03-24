package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrNotificationNotFound             = New(errors.New("notification not found"), http.StatusNotFound, "NOTIFICATION_NOT_FOUND")
	ErrNotificationTitleContentRequired = New(errors.New("title and content are required"), http.StatusBadRequest, "NOTIFICATION_TITLE_CONTENT_REQUIRED")
)
