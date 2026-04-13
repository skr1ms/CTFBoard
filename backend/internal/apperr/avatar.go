package apperr

import "errors"

var (
	ErrInvalidAvatar          = errors.New("invalid avatar file")
	ErrFileTooLarge           = errors.New("avatar file must not exceed 5MB")
	ErrUnsupportedMediaType   = errors.New("supported formats: JPEG, PNG, WebP, GIF")
	ErrAvatarProcessingFailed = errors.New("failed to process avatar image")
	ErrNotTeamCaptain         = errors.New("only team captain can manage team avatar")
)
