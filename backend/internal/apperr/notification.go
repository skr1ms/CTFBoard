package apperr

import "errors"

var (
	ErrNotificationNotFound             = errors.New("notification not found")
	ErrNotificationTitleContentRequired = errors.New("title and content are required")
)
