package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrInvalidAvatar          = New(errors.New("invalid avatar file"), http.StatusBadRequest, "INVALID_AVATAR")
	ErrFileTooLarge           = New(errors.New("avatar file must not exceed 5MB"), http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE")
	ErrUnsupportedMediaType   = New(errors.New("supported formats: JPEG, PNG, WebP, GIF"), http.StatusUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE")
	ErrAvatarProcessingFailed = New(errors.New("failed to process avatar image"), http.StatusInternalServerError, "AVATAR_PROCESSING_FAILED")
	ErrNotTeamCaptain         = New(errors.New("only team captain can manage team avatar"), http.StatusForbidden, "NOT_TEAM_CAPTAIN")
)
