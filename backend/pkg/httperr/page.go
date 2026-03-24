package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrPageNotFound      = New(errors.New("page not found"), http.StatusNotFound, "PAGE_NOT_FOUND")
	ErrPageSlugConflict  = New(errors.New("page slug already exists"), http.StatusConflict, "PAGE_SLUG_CONFLICT")
	ErrPageSlugRequired  = New(errors.New("slug is required"), http.StatusBadRequest, "PAGE_SLUG_REQUIRED")
	ErrPageSlugInvalid   = New(errors.New("invalid page slug format"), http.StatusBadRequest, "PAGE_SLUG_INVALID")
	ErrPageTitleRequired = New(errors.New("title is required"), http.StatusBadRequest, "PAGE_TITLE_REQUIRED")
)
