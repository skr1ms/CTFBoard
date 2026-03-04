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
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/", nil)

	user, ok := RequireUser(w, r)

	assert.False(t, ok)
	assert.Nil(t, user)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
	var body map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&body))
	assert.Equal(t, "not authenticated", body["message"])
}

func TestClampLimit_Nil(t *testing.T) {
	t.Parallel()
	got := ClampLimit(nil, 10, 100)
	assert.Equal(t, 10, got)
}

func TestClampLimit_Zero(t *testing.T) {
	t.Parallel()
	zero := 0
	got := ClampLimit(&zero, 10, 100)
	assert.Equal(t, 10, got)
}

func TestClampLimit_Negative(t *testing.T) {
	t.Parallel()
	neg := -1
	got := ClampLimit(&neg, 10, 100)
	assert.Equal(t, 10, got)
}

func TestClampLimit_WithinRange(t *testing.T) {
	t.Parallel()
	n := 50
	got := ClampLimit(&n, 10, 100)
	assert.Equal(t, 50, got)
}

func TestClampLimit_ExceedsMax(t *testing.T) {
	t.Parallel()
	n := 200
	got := ClampLimit(&n, 10, 100)
	assert.Equal(t, 100, got)
}

func TestParseIntQuery_Missing(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	got := ParseIntQuery(r, "limit")
	assert.Nil(t, got)
}

func TestParseIntQuery_Valid(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/?limit=25", nil)
	got := ParseIntQuery(r, "limit")
	require.NotNil(t, got)
	assert.Equal(t, 25, *got)
}

func TestParseIntQuery_Invalid(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/?limit=abc", nil)
	got := ParseIntQuery(r, "limit")
	assert.Nil(t, got)
}

func TestParseIntQuery_Negative(t *testing.T) {
	t.Parallel()
	r := httptest.NewRequest(http.MethodGet, "/?limit=-5", nil)
	got := ParseIntQuery(r, "limit")
	assert.Nil(t, got)
}
