package httputil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	data := map[string]string{"key": "value"}

	RenderJSON(w, r, http.StatusOK, data)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "value", body["key"])
}

func TestRenderJSON_ErrorStatus(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	RenderJSON(w, r, http.StatusBadRequest, map[string]string{"err": "bad"})

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRenderNoContent_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	RenderNoContent(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Empty(t, w.Body.Bytes())
}

func TestRenderCreated_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/", nil)
	data := map[string]int{"id": 1}

	RenderCreated(w, r, data)

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]int
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, 1, body["id"])
}

func TestRenderOK_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	data := map[string]bool{"ok": true}

	RenderOK(w, r, data)

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]bool
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.True(t, body["ok"])
}

func TestRenderError_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	RenderError(w, r, http.StatusNotFound, "not found")

	assert.Equal(t, http.StatusNotFound, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not found", body["error"])
}

func TestRenderError_Error(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	RenderError(w, r, http.StatusInternalServerError, "internal")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

func TestRenderErrorWithCode_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	RenderErrorWithCode(w, r, http.StatusBadRequest, "invalid", "INVALID_REQUEST")

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid", body["error"])
	assert.Equal(t, "INVALID_REQUEST", body["code"])
}

func TestRenderErrorWithCode_Error(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	RenderErrorWithCode(w, r, http.StatusForbidden, "forbidden", "FORBIDDEN")
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRenderInvalidID_Success(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	RenderInvalidID(w, r)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "invalid ID", body["error"])
}
