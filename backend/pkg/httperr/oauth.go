package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrOAuthAccountNotFound      = New(errors.New("oauth account not found"), http.StatusNotFound, "OAUTH_ACCOUNT_NOT_FOUND")
	ErrOAuthProviderDisabled     = New(errors.New("oauth provider is not enabled"), http.StatusBadRequest, "OAUTH_PROVIDER_DISABLED")
	ErrOAuthUnsupportedProvider  = New(errors.New("unsupported oauth provider"), http.StatusBadRequest, "OAUTH_UNSUPPORTED_PROVIDER")
	ErrOAuthStateMissing         = New(errors.New("missing oauth state cookie"), http.StatusBadRequest, "OAUTH_STATE_MISSING")
	ErrOAuthStateMismatch        = New(errors.New("oauth state mismatch"), http.StatusBadRequest, "OAUTH_STATE_MISMATCH")
	ErrOAuthAccountAlreadyLinked = New(errors.New("oauth account already linked to another user"), http.StatusConflict, "OAUTH_ACCOUNT_ALREADY_LINKED")
)
