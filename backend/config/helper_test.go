package config

import (
	"errors"
	"os"
	"testing"

	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockVaultClient struct {
	secrets map[string]map[string]any
	err     error
}

func (m *mockVaultClient) GetSecret(path string) (map[string]any, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.secrets[path], nil
}

func TestGetEnv_Success(t *testing.T) {
	key := "TEST_GETENV_SUCCESS"
	os.Setenv(key, "  value  ")
	t.Cleanup(func() { os.Unsetenv(key) })

	got := getEnv(key, "default")
	assert.Equal(t, "value", got)
}

func TestGetEnv_Fallback(t *testing.T) {
	got := getEnv("TEST_GETENV_NONEXISTENT_XYZ", "fallback")
	assert.Equal(t, "fallback", got)
}

func TestGetEnvInt_Success(t *testing.T) {
	key := "TEST_GETENVINT_SUCCESS"
	os.Setenv(key, "42")
	t.Cleanup(func() { os.Unsetenv(key) })

	got := getEnvInt(key, 0)
	assert.Equal(t, 42, got)
}

func TestGetEnvInt_Invalid(t *testing.T) {
	key := "TEST_GETENVINT_INVALID"
	os.Setenv(key, "notanint")
	t.Cleanup(func() { os.Unsetenv(key) })

	got := getEnvInt(key, 99)
	assert.Equal(t, 99, got)
}

func TestGetEnvBool_Success(t *testing.T) {
	key := "TEST_GETENVBOOL_SUCCESS"
	os.Setenv(key, "true")
	t.Cleanup(func() { os.Unsetenv(key) })

	got := getEnvBool(key, false)
	assert.True(t, got)
}

func TestGetEnvBool_Invalid(t *testing.T) {
	key := "TEST_GETENVBOOL_INVALID"
	os.Setenv(key, "notabool")
	t.Cleanup(func() { os.Unsetenv(key) })

	got := getEnvBool(key, true)
	assert.True(t, got)
}

func TestParseCORSOrigins_Success(t *testing.T) {
	got := parseCORSOrigins("http://a.com, http://b.com ,https://c.com")
	assert.Equal(t, []string{"http://a.com", "http://b.com", "https://c.com"}, got)
}

func TestParseCORSOrigins_Empty(t *testing.T) {
	got := parseCORSOrigins("")
	assert.Empty(t, got)
}

func TestVaultFetch_Success(t *testing.T) {
	applied := false
	client := &mockVaultClient{
		secrets: map[string]map[string]any{
			"test/path": {"key": "value"},
		},
	}
	l := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})
	fn := vaultFetch(client, l, "test/path", "test", "fallback", func(s map[string]any) {
		applied = true
		assert.Equal(t, "value", s["key"])
	})
	err := fn()
	require.NoError(t, err)
	assert.True(t, applied)
}

func TestVaultFetch_Error(t *testing.T) {
	applied := false
	client := &mockVaultClient{err: errors.New("vault error")}
	l := logger.New(&logger.Options{Level: logger.InfoLevel, Output: logger.ConsoleOutput})
	fn := vaultFetch(client, l, "test/path", "test", "fallback", func(map[string]any) {
		applied = true
	})
	err := fn()
	require.NoError(t, err)
	assert.False(t, applied)
}
