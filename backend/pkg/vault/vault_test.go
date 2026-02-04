package vault

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFromEnv_Error(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")
	t.Cleanup(func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
	})
	_, err := NewFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VAULT_ADDR")
}

func TestNewFromEnv_ErrorNoToken(t *testing.T) {
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	os.Unsetenv("VAULT_TOKEN")
	t.Cleanup(func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
	})
	_, err := NewFromEnv()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VAULT_TOKEN")
}

func TestNewFromEnv_WithCustomMountPath(t *testing.T) {
	os.Setenv("VAULT_ADDR", "http://localhost:8200")
	os.Setenv("VAULT_TOKEN", "token")
	os.Setenv("VAULT_MOUNT_PATH", "custom-mount")
	t.Cleanup(func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
		os.Unsetenv("VAULT_MOUNT_PATH")
	})
	c, err := NewFromEnv()
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestNew_NewWithMount(t *testing.T) {
	c, err := New("http://localhost:8200", "token")
	require.NoError(t, err)
	assert.NotNil(t, c)
}

func TestClient_GetSecret_GetString_Success(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"data": map[string]any{
				"key1": "value1",
				"key2": "value2",
			},
			"metadata": map[string]any{"version": 1},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := NewWithMount(srv.URL, "token", "secret")
	require.NoError(t, err)

	data, err := c.GetSecret("test/path")
	require.NoError(t, err)
	assert.Equal(t, "value1", data["key1"])
	assert.Equal(t, "value2", data["key2"])

	s, err := c.GetString("test/path", "key1")
	require.NoError(t, err)
	assert.Equal(t, "value1", s)
}

func TestClient_GetSecret_Error(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]any{"errors": []string{"no such path"}})
	}))
	defer srv.Close()

	c, err := NewWithMount(srv.URL, "token", "secret")
	require.NoError(t, err)

	_, err = c.GetSecret("missing")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read secret")
}

func TestClient_GetSecret_EmptyData(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"data":     nil,
			"metadata": map[string]any{},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := NewWithMount(srv.URL, "token", "secret")
	require.NoError(t, err)

	_, err = c.GetSecret("empty")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "secret not found")
}

func TestClient_GetString_KeyNotFound(t *testing.T) {
	resp := map[string]any{
		"data": map[string]any{
			"data":     map[string]any{"other": "x"},
			"metadata": map[string]any{},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer srv.Close()

	c, err := NewWithMount(srv.URL, "token", "secret")
	require.NoError(t, err)

	_, err = c.GetString("test", "missing_key")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
