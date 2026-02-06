package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for k, v := range env {
		os.Setenv(k, v)
	}
	t.Cleanup(func() {
		for k := range env {
			os.Unsetenv(k)
		}
	})
}

func TestNew_Success(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")
	t.Cleanup(func() {
		os.Unsetenv("VAULT_ADDR")
		os.Unsetenv("VAULT_TOKEN")
	})

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   "jwt_acc",
		"JWT_REFRESH_SECRET":  "jwt_ref",
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": "flagkey",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.Contains(t, cfg.URL, "u")
	assert.Contains(t, cfg.URL, "d")
	assert.Equal(t, "jwt_acc", cfg.AccessSecret)
	assert.Equal(t, "redis_pwd", cfg.Redis.Password)
	assert.Equal(t, "flagkey", cfg.FlagEncryptionKey)
}

func TestNew_Error_MissingPostgres(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "",
		"POSTGRES_PASSWORD":   "",
		"POSTGRES_DB":         "",
		"JWT_ACCESS_SECRET":   "a",
		"JWT_REFRESH_SECRET":  "r",
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": "fk",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database configuration is missing")
}

func TestNew_Error_MissingJWT(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   "",
		"JWT_REFRESH_SECRET":  "",
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": "fk",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt configuration is missing")
}

func TestNew_Error_MissingRedis(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   "a",
		"JWT_REFRESH_SECRET":  "r",
		"REDIS_PASSWORD":      "",
		"FLAG_ENCRYPTION_KEY": "fk",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis configuration is missing")
}

func TestNew_Error_MissingFlagKey(t *testing.T) {
	os.Unsetenv("VAULT_ADDR")
	os.Unsetenv("VAULT_TOKEN")

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   "a",
		"JWT_REFRESH_SECRET":  "r",
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": "",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flag encryption key")
}
