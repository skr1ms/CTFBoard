package errmap

import (
	"errors"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-httpkit/httperr"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
)

func requireHTTPError(t *testing.T, err error) *httperr.HTTPError {
	t.Helper()

	var he *httperr.HTTPError
	require.ErrorAs(t, err, &he)

	return he
}

func TestMapAppErrorNil(t *testing.T) {
	t.Parallel()

	assert.NoError(t, MapAppError(nil))
}

func TestMapAppErrorPassesHTTPErrorThrough(t *testing.T) {
	t.Parallel()

	original := httperr.New(errors.New("custom"), http.StatusTeapot, "CUSTOM")

	got := MapAppError(original)

	assert.Same(t, original, got)
}

func TestMapAppErrorValidationError(t *testing.T) {
	t.Parallel()

	err := apperr.NewValidationErrorf("invalid %s", "search query")

	got := requireHTTPError(t, MapAppError(err))

	assert.Equal(t, http.StatusBadRequest, got.HTTPStatus())
	assert.Equal(t, "VALIDATION_ERROR", got.GetCode())
	assert.ErrorIs(t, got, err)
}

func TestMapAppErrorKnownSentinels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		status int
		code   string
	}{
		{
			name:   "not found sentinel",
			err:    apperr.ErrUserNotFound,
			status: http.StatusNotFound,
			code:   "USER_NOT_FOUND",
		},
		{
			name:   "wrapped forbidden sentinel",
			err:    fmt.Errorf("service failed: %w", apperr.ErrAccessDenied),
			status: http.StatusForbidden,
			code:   "ACCESS_DENIED",
		},
		{
			name:   "hidden resource maps to generic not found",
			err:    fmt.Errorf("guard: %w", apperr.ErrVisibilityForbidden),
			status: http.StatusNotFound,
			code:   "NOT_FOUND",
		},
		{
			name:   "setup required remains service unavailable",
			err:    apperr.ErrSetupRequired,
			status: http.StatusServiceUnavailable,
			code:   "SETUP_REQUIRED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := requireHTTPError(t, MapAppError(tt.err))

			assert.Equal(t, tt.status, got.HTTPStatus())
			assert.Equal(t, tt.code, got.GetCode())
			assert.ErrorIs(t, got, errors.Unwrap(fmt.Errorf("wrap: %w", tt.err)))
		})
	}
}

func TestMapAppErrorUnknownError(t *testing.T) {
	t.Parallel()

	err := errors.New("boom")

	got := requireHTTPError(t, MapAppError(err))

	assert.Equal(t, http.StatusInternalServerError, got.HTTPStatus())
	assert.Equal(t, "INTERNAL_ERROR", got.GetCode())
	assert.ErrorIs(t, got, err)
}

func TestNewHTTPError(t *testing.T) {
	t.Parallel()

	err := errors.New("bad request")

	got := requireHTTPError(t, NewHTTPError(err, http.StatusBadRequest, "BAD_REQUEST"))

	assert.Equal(t, http.StatusBadRequest, got.HTTPStatus())
	assert.Equal(t, "BAD_REQUEST", got.GetCode())
	assert.ErrorIs(t, got, err)
}

func TestCode(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "TEAM_NOT_FOUND", Code(fmt.Errorf("lookup: %w", apperr.ErrTeamNotFound)))
	assert.Equal(t, "VALIDATION_ERROR", Code(apperr.NewValidationErrorf("bad input")))
	assert.Equal(t, "INTERNAL_ERROR", Code(errors.New("unknown")))
}
