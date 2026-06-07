package helper

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newRequest(method, target string, body io.Reader) *http.Request {
	return httptest.NewRequestWithContext(context.Background(), method, target, body)
}

func TestRequireUser_NoUser(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := newRequest(http.MethodGet, "/", http.NoBody)

	user, ok := RequireUser(w, r)

	assert.False(t, ok)
	assert.Nil(t, user)
	assert.Equal(t, http.StatusUnauthorized, w.Code)

	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not authenticated", body["message"])
}
