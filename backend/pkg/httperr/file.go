package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrFileNotFound        = New(errors.New("file not found"), http.StatusNotFound, "FILE_NOT_FOUND")
	ErrWriteupAccessDenied = New(errors.New("writeup access denied: solve the challenge first"), http.StatusForbidden, "WRITEUP_ACCESS_DENIED")
	ErrWriteupsDisabled    = New(errors.New("writeups are disabled"), http.StatusForbidden, "WRITEUPS_DISABLED")
	ErrFileIDMismatch      = New(errors.New("file ID mismatch"), http.StatusBadRequest, "FILE_ID_MISMATCH")
	ErrFileInvalidToken    = New(errors.New("invalid file token"), http.StatusBadRequest, "FILE_INVALID_TOKEN")
	ErrFileTokenExpired    = New(errors.New("file token expired"), http.StatusBadRequest, "FILE_TOKEN_EXPIRED")
)
