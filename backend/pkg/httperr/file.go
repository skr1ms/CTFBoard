package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrFileNotFound = &HTTPError{
		Err:        errors.New("file not found"),
		StatusCode: http.StatusNotFound,
		Code:       "FILE_NOT_FOUND",
	}
	ErrWriteupAccessDenied = &HTTPError{
		Err:        errors.New("writeup access denied: solve the challenge first"),
		StatusCode: http.StatusForbidden,
		Code:       "WRITEUP_ACCESS_DENIED",
	}
	ErrWriteupsDisabled = &HTTPError{
		Err:        errors.New("writeups are disabled"),
		StatusCode: http.StatusForbidden,
		Code:       "WRITEUPS_DISABLED",
	}
	ErrFileIDMismatch = &HTTPError{
		Err:        errors.New("file ID mismatch"),
		StatusCode: http.StatusBadRequest,
		Code:       "FILE_ID_MISMATCH",
	}
	ErrFileInvalidToken = &HTTPError{
		Err:        errors.New("invalid file token"),
		StatusCode: http.StatusBadRequest,
		Code:       "FILE_INVALID_TOKEN",
	}
	ErrFileTokenExpired = &HTTPError{
		Err:        errors.New("file token expired"),
		StatusCode: http.StatusBadRequest,
		Code:       "FILE_TOKEN_EXPIRED",
	}
)
