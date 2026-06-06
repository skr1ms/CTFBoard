package config

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// validate performs cross-field validation of rawConfig after env + Vault loading.
// It checks: required Postgres/JWT/Redis credentials are present; JWT secrets meet
// the minimum length required by go-jwtkit; FLAG_ENCRYPTION_KEY is exactly 64 hex
// chars (32 bytes AES-256); partial OAuth provider configuration fails fast;
// COMPETITION_MODE is a recognized value; MIN/MAX_TEAM_SIZE range is valid;
// STORAGE_PROVIDER is one of filesystem or s3; TTLs and rate limits are positive.
func validate(raw *rawConfig) error {
	if raw.PostgresUser == "" || raw.PostgresPassword == "" || raw.PostgresDB == "" {
		return fmt.Errorf("required database configuration is missing (env or vault)")
	}

	if raw.JWTAccessSecret == "" || raw.JWTRefreshSecret == "" {
		return fmt.Errorf("required jwt configuration is missing (env or vault)")
	}

	if len(raw.JWTAccessSecret) < jwtkit.MinSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d bytes, got %d", jwtkit.MinSecretLength, len(raw.JWTAccessSecret))
	}

	if len(raw.JWTRefreshSecret) < jwtkit.MinSecretLength {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least %d bytes, got %d", jwtkit.MinSecretLength, len(raw.JWTRefreshSecret))
	}

	if raw.RedisPassword == "" {
		return fmt.Errorf("required redis configuration is missing (env or vault)")
	}

	if raw.FlagEncryptionKey == "" {
		return fmt.Errorf("required flag encryption key is missing (env or vault) - needed for regex challenges")
	}

	if len(raw.FlagEncryptionKey) != flagEncryptionKeyHexLen {
		return fmt.Errorf("FLAG_ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes for AES-256), got %d", len(raw.FlagEncryptionKey))
	}

	if _, err := hex.DecodeString(raw.FlagEncryptionKey); err != nil {
		return fmt.Errorf("FLAG_ENCRYPTION_KEY contains invalid hex characters: %w", err)
	}

	if raw.SetupToken != "" && len(raw.SetupToken) < 32 {
		return fmt.Errorf("SETUP_TOKEN must be at least 32 characters when set")
	}

	if err := validateOAuthProvider("github", raw.OAuthGitHubClientID, raw.OAuthGitHubClientSecret, raw.OAuthGitHubRedirectURL); err != nil {
		return err
	}

	if err := validateOAuthProvider("google", raw.OAuthGoogleClientID, raw.OAuthGoogleClientSecret, raw.OAuthGoogleRedirectURL); err != nil {
		return err
	}

	if (raw.OAuthGitHubClientID != "" || raw.OAuthGitHubClientSecret != "" || raw.OAuthGoogleClientID != "" || raw.OAuthGoogleClientSecret != "") && raw.OAuthStateSecret == "" {
		return fmt.Errorf("OAUTH_STATE_SECRET is required when OAuth clients are configured")
	}

	if !domain.CompetitionMode(raw.CompetitionMode).IsValid() {
		return fmt.Errorf("invalid COMPETITION_MODE %q: must be solo_only, teams_only, or flexible", raw.CompetitionMode)
	}

	if raw.MinTeamSize < 1 || raw.MaxTeamSize < raw.MinTeamSize {
		return fmt.Errorf("invalid team size range: MIN_TEAM_SIZE=%d must be >= 1 and <= MAX_TEAM_SIZE=%d", raw.MinTeamSize, raw.MaxTeamSize)
	}

	switch raw.StorageProvider {
	case "filesystem", "s3":
	default:
		return fmt.Errorf("invalid STORAGE_PROVIDER %q: must be filesystem or s3", raw.StorageProvider)
	}

	if raw.RateLimitSubmitFlag <= 0 {
		return fmt.Errorf("RATE_LIMIT_SUBMIT_FLAG must be a positive integer, got %d", raw.RateLimitSubmitFlag)
	}

	if raw.RateLimitSubmitFlagDuration <= 0 {
		return fmt.Errorf("RATE_LIMIT_SUBMIT_FLAG_DURATION must be a positive integer, got %d", raw.RateLimitSubmitFlagDuration)
	}

	if raw.JWTAccessTTLMin <= 0 {
		return fmt.Errorf("JWT_ACCESS_TTL_MINUTES must be a positive integer, got %d", raw.JWTAccessTTLMin)
	}

	if raw.JWTRefreshTTLHrs <= 0 {
		return fmt.Errorf("JWT_REFRESH_TTL_HOURS must be a positive integer, got %d", raw.JWTRefreshTTLHrs)
	}

	if raw.ResendVerifyTTLHrs <= 0 {
		return fmt.Errorf("RESEND_VERIFY_TTL_HOURS must be a positive integer, got %d", raw.ResendVerifyTTLHrs)
	}

	if raw.ResendResetTTLHrs <= 0 {
		return fmt.Errorf("RESEND_RESET_TTL_HOURS must be a positive integer, got %d", raw.ResendResetTTLHrs)
	}

	if raw.StoragePresignedExpiryMin <= 0 {
		return fmt.Errorf("STORAGE_PRESIGNED_EXPIRY_MINUTES must be a positive integer, got %d", raw.StoragePresignedExpiryMin)
	}

	return nil
}

func validateOAuthProvider(name, clientID, clientSecret, redirectURL string) error {
	if clientID == "" && clientSecret == "" {
		return nil
	}

	if clientID == "" || clientSecret == "" || redirectURL == "" {
		return fmt.Errorf("OAUTH_%s_CLIENT_ID, OAUTH_%s_CLIENT_SECRET and OAUTH_%s_REDIRECT_URL must all be set to enable %s OAuth",
			strings.ToUpper(name), strings.ToUpper(name), strings.ToUpper(name), name)
	}

	return nil
}

func isUsableResendAPIKey(apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)

	return apiKey != "" && apiKey != "placeholder"
}
