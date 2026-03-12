package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrPageNotFound = &HTTPError{
		Err:        errors.New("page not found"),
		StatusCode: http.StatusNotFound,
		Code:       "PAGE_NOT_FOUND",
	}
	ErrPageSlugConflict = &HTTPError{
		Err:        errors.New("page slug already exists"),
		StatusCode: http.StatusConflict,
		Code:       "PAGE_SLUG_CONFLICT",
	}
	ErrPageSlugRequired = &HTTPError{
		Err:        errors.New("slug is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "PAGE_SLUG_REQUIRED",
	}
	ErrPageSlugInvalid = &HTTPError{
		Err:        errors.New("invalid page slug format"),
		StatusCode: http.StatusBadRequest,
		Code:       "PAGE_SLUG_INVALID",
	}
	ErrPageTitleRequired = &HTTPError{
		Err:        errors.New("title is required"),
		StatusCode: http.StatusBadRequest,
		Code:       "PAGE_TITLE_REQUIRED",
	}
)
