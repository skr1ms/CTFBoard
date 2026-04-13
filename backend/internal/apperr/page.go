package apperr

import "errors"

var (
	ErrPageNotFound      = errors.New("page not found")
	ErrPageSlugConflict  = errors.New("page slug already exists")
	ErrPageSlugRequired  = errors.New("slug is required")
	ErrPageSlugInvalid   = errors.New("invalid page slug format")
	ErrPageTitleRequired = errors.New("title is required")
)
