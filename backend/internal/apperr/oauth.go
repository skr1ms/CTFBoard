package apperr

import "errors"

var (
	ErrOAuthAccountNotFound      = errors.New("oauth account not found")
	ErrOAuthProviderDisabled     = errors.New("oauth provider is not enabled")
	ErrOAuthUnsupportedProvider  = errors.New("unsupported oauth provider")
	ErrOAuthStateMissing         = errors.New("missing oauth state cookie")
	ErrOAuthStateMismatch        = errors.New("oauth state mismatch")
	ErrOAuthAccountAlreadyLinked = errors.New("oauth account already linked to another user")
)
