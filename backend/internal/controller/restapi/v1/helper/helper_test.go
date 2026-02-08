package helper

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequireUser_NoUser(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	user, ok := RequireUser(w, r)

	assert.False(t, ok)
	assert.Nil(t, user)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]interface{}
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not authenticated", body["error"])
}
