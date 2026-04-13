package apperr

import "errors"

var (
	ErrFileNotFound         = errors.New("file not found")
	ErrFileLocationConflict = errors.New("file with this location already exists")
	ErrWriteupAccessDenied  = errors.New("writeup access denied: solve the challenge first")
	ErrWriteupsDisabled     = errors.New("writeups are disabled")
	ErrFileIDMismatch       = errors.New("file ID mismatch")
	ErrFileInvalidToken     = errors.New("invalid file token")
	ErrFileTokenExpired     = errors.New("file token expired")
)
