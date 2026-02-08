package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderOK(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	RenderOK(w, r, map[string]string{"k": "v"})

	assert.Equal(t, http.StatusOK, w.Code)
	var body map[string]string
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "v", body["k"])
}

func TestRenderCreated(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", nil)

	RenderCreated(w, r, map[string]int{"id": 42})

	assert.Equal(t, http.StatusCreated, w.Code)
	var body map[string]int
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, 42, body["id"])
}

func TestRenderNoContent(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodDelete, "/", nil)

	RenderNoContent(w, r)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, 0, w.Body.Len())
}

func TestRenderJSON(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	RenderJSON(w, r, http.StatusAccepted, map[string]bool{"ok": true})

	assert.Equal(t, http.StatusAccepted, w.Code)
	var body map[string]bool
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.True(t, body["ok"])
}
