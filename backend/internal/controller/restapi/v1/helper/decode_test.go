package helper

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/validator"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDecodeAndValidateE_InvalidJSON(t *testing.T) {
	t.Parallel()
	body := bytes.NewReader([]byte("not json"))
	r := httptest.NewRequest(http.MethodPost, "/", body)
	v, err := validator.New()
	require.NoError(t, err)

	_, err = DecodeAndValidateE[struct{ X int }](r, v)
	require.Error(t, err)
	var httpErr *httperr.HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
	assert.Equal(t, "INVALID_JSON", httpErr.Code)
}

func TestDecodeAndValidateE_ValidationError(t *testing.T) {
	t.Parallel()
	type S struct {
		Email string `json:"email" validate:"required,email"`
	}
	body := bytes.NewReader([]byte(`{"email":"invalid"}`))
	r := httptest.NewRequest(http.MethodPost, "/", body)
	v, err := validator.New()
	require.NoError(t, err)

	_, err = DecodeAndValidateE[S](r, v)
	require.Error(t, err)
	var httpErr *httperr.HTTPError
	require.True(t, errors.As(err, &httpErr))
	assert.Equal(t, http.StatusBadRequest, httpErr.StatusCode)
	assert.Equal(t, "VALIDATION_ERROR", httpErr.Code)
}

func TestDecodeAndValidateE_Ok(t *testing.T) {
	t.Parallel()
	type S struct {
		Email string `json:"email" validate:"required,email"`
	}
	body := bytes.NewReader([]byte(`{"email":"a@b.c"}`))
	r := httptest.NewRequest(http.MethodPost, "/", body)
	v, err := validator.New()
	require.NoError(t, err)

	out, err := DecodeAndValidateE[S](r, v)
	require.NoError(t, err)
	assert.Equal(t, "a@b.c", out.Email)
}

func TestParseUUID_Empty(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	parsed, ok := ParseUUID(w, r, "")
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, parsed)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseUUID_Invalid(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	parsed, ok := ParseUUID(w, r, "not-a-uuid")
	assert.False(t, ok)
	assert.Equal(t, uuid.Nil, parsed)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestParseUUID_Valid(t *testing.T) {
	t.Parallel()
	id := uuid.New()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	parsed, ok := ParseUUID(w, r, id.String())
	assert.True(t, ok)
	assert.Equal(t, id, parsed)
}

func TestGetClientIP(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "192.168.1.1:12345"
	assert.Equal(t, "192.168.1.1", GetClientIP(r, nil))

	r.Header.Set("X-Real-IP", "10.0.0.1")
	r.RemoteAddr = "127.0.0.1:80"
	assert.Equal(t, "10.0.0.1", GetClientIP(r, []string{"127.0.0.0/8"}))
}
