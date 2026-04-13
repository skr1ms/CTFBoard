package apperr

import "errors"

var (
	ErrUserNotFound                = errors.New("user not found")
	ErrUserAlreadyExists           = errors.New("user already exists")
	ErrUsernameTaken               = errors.New("username already taken")
	ErrRegistrationClosed          = errors.New("registration is closed")
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrAuthorizationHeaderRequired = errors.New("authorization header required")
	ErrInvalidAuthorizationHeader  = errors.New("invalid authorization header format")
	ErrInvalidToken                = errors.New("invalid token")
	ErrAccessDenied                = errors.New("access denied")
	ErrUserBanned                  = errors.New("user is banned")
	ErrCaptainCannotBeDeleted      = errors.New("transfer captain first, then delete user")
)
