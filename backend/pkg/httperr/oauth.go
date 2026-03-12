package httperr

import (
	"errors"
	"net/http"
)

var (
	ErrOAuthAccountNotFound = &HTTPError{
		Err:        errors.New("oauth account not found"),
		StatusCode: http.StatusNotFound,
		Code:       "OAUTH_ACCOUNT_NOT_FOUND",
	}
	ErrOAuthProviderDisabled = &HTTPError{
		Err:        errors.New("oauth provider is not enabled"),
		StatusCode: http.StatusBadRequest,
		Code:       "OAUTH_PROVIDER_DISABLED",
	}
	ErrOAuthUnsupportedProvider = &HTTPError{
		Err:        errors.New("unsupported oauth provider"),
		StatusCode: http.StatusBadRequest,
		Code:       "OAUTH_UNSUPPORTED_PROVIDER",
	}
	ErrOAuthStateMissing = &HTTPError{
		Err:        errors.New("missing oauth state cookie"),
		StatusCode: http.StatusBadRequest,
		Code:       "OAUTH_STATE_MISSING",
	}
	ErrOAuthStateMismatch = &HTTPError{
		Err:        errors.New("oauth state mismatch"),
		StatusCode: http.StatusBadRequest,
		Code:       "OAUTH_STATE_MISMATCH",
	}

	ErrOAuthAccountAlreadyLinked = &HTTPError{
		Err:        errors.New("oauth account already linked to another user"),
		StatusCode: http.StatusConflict,
		Code:       "OAUTH_ACCOUNT_ALREADY_LINKED",
	}
)
