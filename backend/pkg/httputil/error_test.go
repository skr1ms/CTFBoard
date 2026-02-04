package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	entityError "github.com/skr1ms/CTFBoard/internal/entity/error"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandleError_HTTPError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	HandleError(w, r, entityError.ErrUserNotFound)
	assert.Equal(t, http.StatusNotFound, w.Code)
	var body ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "USER_NOT_FOUND", body.Code)
}

func TestHandleError_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	HandleError(w, r, assert.AnError)
	assert.Equal(t, http.StatusInternalServerError, w.Code)
	var body ErrorResponse
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "Internal server error", body.Error)
}
