package config

import (
	"encoding/hex"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const minSetupTokenLen = 32

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

	if raw.SetupToken == "" {
		return fmt.Errorf("SETUP_TOKEN must be set")
	}

	if len(raw.SetupToken) < minSetupTokenLen {
		return fmt.Errorf("SETUP_TOKEN must be at least %d characters", minSetupTokenLen)
	}

	if err := validateCORSOrigins(raw.CORSOrigins); err != nil {
		return err
	}

	if err := validateTrustedProxyCIDRs(raw.TrustedProxyCIDRs); err != nil {
		return err
	}

	if err := validatePort("BACKEND_PORT", raw.BackendPort); err != nil {
		return err
	}

	if err := validatePort("POSTGRES_PORT", raw.PostgresPort); err != nil {
		return err
	}

	if err := validatePort("REDIS_PORT", raw.RedisPort); err != nil {
		return err
	}

	if raw.PostgresMaxConns <= 0 {
		return fmt.Errorf("POSTGRES_MAX_CONNS must be a positive integer, got %d", raw.PostgresMaxConns)
	}

	if raw.PostgresMinConns < 0 || raw.PostgresMinConns > raw.PostgresMaxConns {
		return fmt.Errorf("POSTGRES_MIN_CONNS=%d must be >= 0 and <= POSTGRES_MAX_CONNS=%d", raw.PostgresMinConns, raw.PostgresMaxConns)
	}

	if raw.RedisPoolSize <= 0 {
		return fmt.Errorf("REDIS_POOL_SIZE must be a positive integer, got %d", raw.RedisPoolSize)
	}

	if raw.RedisMinIdle < 0 || raw.RedisMinIdle > raw.RedisPoolSize {
		return fmt.Errorf("REDIS_MIN_IDLE=%d must be >= 0 and <= REDIS_POOL_SIZE=%d", raw.RedisMinIdle, raw.RedisPoolSize)
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
		return fmt.Errorf("invalid COMPETITION_MODE %q: must be solo_only or teams_only", raw.CompetitionMode)
	}

	if raw.MinTeamSize < 1 || raw.MaxTeamSize < raw.MinTeamSize {
		return fmt.Errorf("invalid team size range: MIN_TEAM_SIZE=%d must be >= 1 and <= MAX_TEAM_SIZE=%d", raw.MinTeamSize, raw.MaxTeamSize)
	}

	switch raw.StorageProvider {
	case "filesystem", "s3":
	default:
		return fmt.Errorf("invalid STORAGE_PROVIDER %q: must be filesystem or s3", raw.StorageProvider)
	}

	if raw.StorageProvider == "filesystem" && strings.TrimSpace(raw.StorageLocalPath) == "" {
		return fmt.Errorf("STORAGE_LOCAL_PATH must be set when STORAGE_PROVIDER=filesystem")
	}

	if raw.S3PublicEndpoint != "" {
		if err := validateExactHTTPOrigin("STORAGE_S3_PUBLIC_ENDPOINT", raw.S3PublicEndpoint); err != nil {
			return err
		}
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

func validatePort(name, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil {
		return fmt.Errorf("%s must be a numeric TCP port: %w", name, err)
	}

	if port < 1 || port > 65535 {
		return fmt.Errorf("%s must be between 1 and 65535, got %d", name, port)
	}

	return nil
}

func validateHTTPURL(name, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s contains invalid URL %q: %w", name, value, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https scheme", name)
	}

	if u.Host == "" {
		return fmt.Errorf("%s must include a host", name)
	}

	return nil
}

func validateExactHTTPOrigin(name, value string) error {
	u, err := url.Parse(value)
	if err != nil {
		return fmt.Errorf("%s contains invalid URL %q: %w", name, value, err)
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("%s must use http or https scheme", name)
	}

	if u.Host == "" {
		return fmt.Errorf("%s must include a host", name)
	}

	if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("%s must be an exact origin without user info, path, query, or fragment", name)
	}

	return nil
}

func validateTrustedProxyCIDRs(cidrs []string) error {
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("TRUSTED_PROXY_CIDRS contains invalid CIDR %q: %w", cidr, err)
		}

		ones, _ := network.Mask.Size()
		if ones == 0 {
			return fmt.Errorf("TRUSTED_PROXY_CIDRS must not trust all addresses via %q", cidr)
		}
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

	return validateHTTPURL("OAUTH_"+strings.ToUpper(name)+"_REDIRECT_URL", redirectURL)
}

func validateCORSOrigins(origins []string) error {
	for _, origin := range origins {
		if origin == "*" || strings.Contains(origin, "*") {
			return fmt.Errorf("CORS_ORIGINS must not contain wildcard origin %q when credentialed CORS is enabled", origin)
		}

		u, err := url.Parse(origin)
		if err != nil {
			return fmt.Errorf("CORS_ORIGINS contains invalid origin %q: %w", origin, err)
		}

		if u.Scheme != "http" && u.Scheme != "https" {
			return fmt.Errorf("CORS_ORIGINS origin %q must use http or https scheme", origin)
		}

		if u.Host == "" {
			return fmt.Errorf("CORS_ORIGINS origin %q must include a host", origin)
		}

		if u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" {
			return fmt.Errorf("CORS_ORIGINS origin %q must be an exact origin without user info, path, query, or fragment", origin)
		}
	}

	return nil
}

func isUsableResendAPIKey(apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)

	return apiKey != "" && apiKey != "placeholder"
}
