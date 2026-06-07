package config

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const jwtSecret32 = "12345678901234567890123456789012"

const flagKey64Hex = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

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

func disableVaultForTest(t *testing.T) {
	t.Helper()
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")
}

func TestNew_Success(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"RESEND_ENABLED":      "false",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.Contains(t, cfg.URL, "u")
	assert.Contains(t, cfg.URL, "d")
	assert.Equal(t, jwtSecret32, cfg.AccessSecret)
	assert.Equal(t, "redis_pwd", cfg.Redis.Password)
	assert.Equal(t, flagKey64Hex, cfg.FlagEncryptionKey)
}

func TestNew_Error_MissingPostgres(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "",
		"POSTGRES_PASSWORD":   "",
		"POSTGRES_DB":         "",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "database configuration is missing")
}

func TestNew_Error_MissingJWT(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   "",
		"JWT_REFRESH_SECRET":  "",
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "jwt configuration is missing")
}

func TestNew_Error_MissingRedis(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis configuration is missing")
}

func TestNew_Error_MissingFlagKey(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": "",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "flag encryption key")
}

func TestNew_Error_FlexibleCompetitionMode(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "rp",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"COMPETITION_MODE":    "flexible",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be solo_only or teams_only")
}

func TestNew_ShutdownTimeout_Default(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"RESEND_ENABLED":      "false",
	})
	// HTTP_SHUTDOWN_TIMEOUT not set -> default 15s
	cfg, err := New()
	require.NoError(t, err)
	assert.Equal(t, 15*time.Second, cfg.ShutdownTimeout)
}

func TestNew_ShutdownTimeout_Custom(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":         "u",
		"POSTGRES_PASSWORD":     "p",
		"POSTGRES_DB":           "d",
		"JWT_ACCESS_SECRET":     jwtSecret32,
		"JWT_REFRESH_SECRET":    jwtSecret32,
		"REDIS_PASSWORD":        "redis_pwd",
		"FLAG_ENCRYPTION_KEY":   flagKey64Hex,
		"RESEND_ENABLED":        "false",
		"HTTP_SHUTDOWN_TIMEOUT": "30",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, cfg.ShutdownTimeout)
}

func TestNew_ResendEnabledNoAPIKey_DisablesEmail(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"RESEND_ENABLED":      "true",
		"RESEND_API_KEY":      "",
		"VERIFY_EMAILS":       "true",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Empty(t, cfg.APIKey)
	assert.False(t, cfg.VerifyEmails)
}

func TestNew_ResendPlaceholder_DisablesEmail(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"RESEND_ENABLED":      "true",
		"RESEND_API_KEY":      "placeholder",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.False(t, cfg.Enabled)
	assert.Equal(t, "placeholder", cfg.APIKey)
}

func TestNew_Error_S3ProviderMissingConfig(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"RESEND_ENABLED":      "false",
		"STORAGE_PROVIDER":    "s3",
		"STORAGE_S3_ENDPOINT": "",
		"STORAGE_S3_BUCKET":   "",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "S3_ENDPOINT")
	assert.Contains(t, err.Error(), "S3_BUCKET")
	assert.Contains(t, err.Error(), "STORAGE_PROVIDER")
}

func TestNew_Error_SetupTokenTooShort(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":       "u",
		"POSTGRES_PASSWORD":   "p",
		"POSTGRES_DB":         "d",
		"JWT_ACCESS_SECRET":   jwtSecret32,
		"JWT_REFRESH_SECRET":  jwtSecret32,
		"REDIS_PASSWORD":      "redis_pwd",
		"FLAG_ENCRYPTION_KEY": flagKey64Hex,
		"SETUP_TOKEN":         "short",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "SETUP_TOKEN")
}

func TestNew_Error_OAuthPartialConfig(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":              "u",
		"POSTGRES_PASSWORD":          "p",
		"POSTGRES_DB":                "d",
		"JWT_ACCESS_SECRET":          jwtSecret32,
		"JWT_REFRESH_SECRET":         jwtSecret32,
		"REDIS_PASSWORD":             "redis_pwd",
		"FLAG_ENCRYPTION_KEY":        flagKey64Hex,
		"OAUTH_STATE_SECRET":         jwtSecret32,
		"OAUTH_GITHUB_CLIENT_ID":     "client",
		"OAUTH_GITHUB_CLIENT_SECRET": "",
		"OAUTH_GITHUB_REDIRECT_URL":  "http://localhost/callback",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "OAUTH_GITHUB_CLIENT_ID")
	assert.Contains(t, err.Error(), "OAUTH_GITHUB_CLIENT_SECRET")
}

func TestNew_OAuthRedirectOnlyDoesNotEnableProvider(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":             "u",
		"POSTGRES_PASSWORD":         "p",
		"POSTGRES_DB":               "d",
		"JWT_ACCESS_SECRET":         jwtSecret32,
		"JWT_REFRESH_SECRET":        jwtSecret32,
		"REDIS_PASSWORD":            "redis_pwd",
		"FLAG_ENCRYPTION_KEY":       flagKey64Hex,
		"OAUTH_GITHUB_REDIRECT_URL": "http://localhost/callback",
	})

	cfg, err := New()
	require.NoError(t, err)
	assert.False(t, cfg.GitHub.IsConfigured())
}

func TestNew_Error_InvalidRefreshTTL(t *testing.T) {
	disableVaultForTest(t)

	setupEnv(t, map[string]string{
		"POSTGRES_USER":         "u",
		"POSTGRES_PASSWORD":     "p",
		"POSTGRES_DB":           "d",
		"JWT_ACCESS_SECRET":     jwtSecret32,
		"JWT_REFRESH_SECRET":    jwtSecret32,
		"REDIS_PASSWORD":        "redis_pwd",
		"FLAG_ENCRYPTION_KEY":   flagKey64Hex,
		"JWT_REFRESH_TTL_HOURS": "0",
	})

	_, err := New()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "JWT_REFRESH_TTL_HOURS")
}
