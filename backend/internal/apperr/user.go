package apperr

import "errors"

var (
	ErrUserNotFound                = errors.New("user not found")
	ErrUserAlreadyExists           = errors.New("user already exists")
	ErrUsernameTaken               = errors.New("username already taken")
	ErrRegistrationClosed          = errors.New("registration is closed")
	ErrRegistrationCodeRequired    = errors.New("registration code is required")
	ErrInvalidRegistrationCode     = errors.New("invalid registration code")
	ErrMaxUsersReached             = errors.New("maximum number of users reached")
	ErrInvalidCredentials          = errors.New("invalid credentials")
	ErrLocalPasswordRequired       = errors.New("local password is required")
	ErrAuthorizationHeaderRequired = errors.New("authorization header required")
	ErrInvalidAuthorizationHeader  = errors.New("invalid authorization header format")
	ErrInvalidToken                = errors.New("invalid token")
	ErrAccessDenied                = errors.New("access denied")
	ErrUserBanned                  = errors.New("user is banned")
	ErrCaptainCannotBeDeleted      = errors.New("transfer captain first, then delete user")
	ErrAppealNotFound              = errors.New("appeal not found")
	ErrAppealRateLimited           = errors.New("appeal rate limited: only one appeal per 7 days")
	ErrAnimatedImageNotAllowed     = errors.New("animated images are not allowed")
)
