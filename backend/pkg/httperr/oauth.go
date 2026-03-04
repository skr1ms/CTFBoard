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
	ErrInvalidFrontendURL = &HTTPError{
		Err:        errors.New("invalid frontend URL"),
		StatusCode: http.StatusInternalServerError,
		Code:       "INTERNAL_ERROR",
	}
	ErrOAuthCodeExchange = &HTTPError{
		Err:        errors.New("oauth code exchange failed"),
		StatusCode: http.StatusBadGateway,
		Code:       "OAUTH_CODE_EXCHANGE_FAILED",
	}
	ErrOAuthAccountAlreadyLinked = &HTTPError{
		Err:        errors.New("oauth account already linked to another user"),
		StatusCode: http.StatusConflict,
		Code:       "OAUTH_ACCOUNT_ALREADY_LINKED",
	}
	ErrOAuthProfileFetch = &HTTPError{
		Err:        errors.New("failed to fetch oauth user profile"),
		StatusCode: http.StatusBadGateway,
		Code:       "OAUTH_PROFILE_FETCH_FAILED",
	}
)
