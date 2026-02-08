package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrInvalidID(t *testing.T) {
	assert.Equal(t, "invalid ID format", ErrInvalidID.Error)
	assert.Equal(t, "INVALID_ID", ErrInvalidID.Code)
}

func TestErrorResponse_Render(t *testing.T) {
	e := &ErrorResponse{Error: "msg", Code: "CODE"}
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	err := e.Render(w, r)
	require.NoError(t, err)
}

func TestRenderInvalidID(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	RenderInvalidID(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid ID format", body["error"])
	assert.Equal(t, "INVALID_ID", body["code"])
}

func TestRenderError(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	RenderError(w, r, http.StatusForbidden, "forbidden")

	assert.Equal(t, http.StatusForbidden, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "forbidden", body["error"])
}

func TestRenderErrorWithCode(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	RenderErrorWithCode(w, r, http.StatusUnprocessableEntity, "validation failed", "VALIDATION")

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "validation failed", body["error"])
	assert.Equal(t, "VALIDATION", body["code"])
}
