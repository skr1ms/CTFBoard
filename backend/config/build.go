package config

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// buildConfig assembles the final Config from a validated rawConfig. Key steps:
// JWT key rotation arrays (JWT_ACCESS_KEYS / JWT_REFRESH_KEYS) are parsed from
// JSON if present, otherwise the single primary secret is wrapped as kid="0".
// If JWT_DOWNLOAD_SECRET is empty, it is derived via HMAC-SHA256 over the access
// secret with the fixed label "download-url-signing". Share-link signing derives
// a separate secret with the fixed label "share-link-signing". The Postgres DSN is
// constructed programmatically to avoid injection via url.URL. SecureCookies is
// forced true when API_BASE_URL starts with https://. Post-assembly warnings are
// emitted for known configuration anti-patterns such as solo_only with MinTeamSize>1.
func buildConfig(raw *rawConfig, l logkit.Logger) (*Config, error) {
	jwtAccessKeys, err := parseJWTKeys("JWT_ACCESS_KEYS", raw.JWTAccessKeysStr, raw.JWTAccessSecret)
	if err != nil {
		return nil, err
	}

	jwtRefreshKeys, err := parseJWTKeys("JWT_REFRESH_KEYS", raw.JWTRefreshKeysStr, raw.JWTRefreshSecret)
	if err != nil {
		return nil, err
	}

	resend := buildResend(raw)

	cfg := &Config{
		App: buildApp(raw, resend.Enabled),
		Admin: Admin{
			Username: raw.AdminUsername,
			Email:    raw.AdminEmail,
			Password: raw.AdminPassword,
		},
		HTTP:        buildHTTP(raw),
		DB:          buildDB(raw),
		JWT:         buildJWT(raw, jwtAccessKeys, jwtRefreshKeys),
		Redis:       buildRedis(raw),
		RateLimit:   buildRateLimit(raw),
		Resend:      resend,
		Storage:     buildStorage(raw),
		Competition: buildCompetition(raw),
		OAuth:       buildOAuth(raw),
	}

	if err := validateStorageConfig(cfg.Storage); err != nil {
		return nil, err
	}

	warnCompetitionConfig(cfg.Competition, l)

	return cfg, nil
}

func parseJWTKeys(envName, rawValue, primarySecret string) ([]JWTKey, error) {
	keys := []JWTKey{{Kid: "0", Secret: primarySecret}}

	if rawValue == "" {
		return keys, nil
	}

	var parsed []JWTKey

	if err := json.Unmarshal([]byte(rawValue), &parsed); err != nil {
		return nil, fmt.Errorf("%s invalid JSON: %w", envName, err)
	}

	if len(parsed) == 0 {
		return nil, fmt.Errorf("%s must contain at least one key", envName)
	}

	for i, k := range parsed {
		if len(k.Secret) < jwtkit.MinSecretLength {
			return nil, fmt.Errorf("%s[%d] secret must be at least %d bytes", envName, i, jwtkit.MinSecretLength)
		}
	}

	return parsed, nil
}

func buildApp(raw *rawConfig, resendEnabled bool) App {
	return App{
		Name:              raw.AppName,
		Version:           raw.AppVersion,
		StructuredLogger:  raw.StructuredLogger,
		SecureCookies:     raw.SecureCookies || strings.HasPrefix(raw.APIBaseURL, "https://"),
		SetupToken:        raw.SetupToken,
		LogLevel:          raw.LogLevel,
		FlagEncryptionKey: raw.FlagEncryptionKey,
		VerifyEmails:      raw.VerifyEmails && resendEnabled,
		DebugEnabled:      raw.DebugEnabled,
	}
}

func buildHTTP(raw *rawConfig) HTTP {
	return HTTP{
		Port:              raw.BackendPort,
		BaseURL:           raw.APIBaseURL,
		CORSOrigins:       raw.CORSOrigins,
		TrustedProxyCIDRs: raw.TrustedProxyCIDRs,
		MetricsAllowedIPs: raw.MetricsAllowedIPs,
		ShutdownTimeout:   time.Duration(raw.ShutdownTimeoutSec) * time.Second,
	}
}

func buildDB(raw *rawConfig) DB {
	return DB{
		URL:            postgresURL(raw),
		MigrationsPath: raw.MigrationsPath,
		MaxConns:       raw.PostgresMaxConns,
		MinConns:       raw.PostgresMinConns,
	}
}

func postgresURL(raw *rawConfig) string {
	return (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(raw.PostgresUser, raw.PostgresPassword),
		Host:     raw.PostgresHost + ":" + raw.PostgresPort,
		Path:     raw.PostgresDB,
		RawQuery: "sslmode=" + url.QueryEscape(raw.DBSSLMode),
	}).String()
}

func buildJWT(raw *rawConfig, accessKeys, refreshKeys []JWTKey) JWT {
	return JWT{
		AccessSecret:   raw.JWTAccessSecret,
		RefreshSecret:  raw.JWTRefreshSecret,
		DownloadSecret: downloadSecret(raw),
		ShareSecret:    shareSecret(raw),
		AccessKeys:     accessKeys,
		RefreshKeys:    refreshKeys,
		AccessTTL:      time.Duration(raw.JWTAccessTTLMin) * time.Minute,
		RefreshTTL:     time.Duration(raw.JWTRefreshTTLHrs) * time.Hour,
		Issuer:         raw.JWTIssuer,
	}
}

func downloadSecret(raw *rawConfig) string {
	if raw.JWTDownloadSecret != "" {
		return raw.JWTDownloadSecret
	}

	return derivedJWTSecret(raw.JWTAccessSecret, "download-url-signing")
}

func shareSecret(raw *rawConfig) string {
	return derivedJWTSecret(raw.JWTAccessSecret, "share-link-signing")
}

func derivedJWTSecret(accessSecret, label string) string {
	h := hmac.New(sha256.New, []byte(accessSecret))
	_, _ = h.Write([]byte(label))

	return hex.EncodeToString(h.Sum(nil))
}

func buildRedis(raw *rawConfig) Redis {
	return Redis{
		Host:         raw.RedisHost,
		Port:         raw.RedisPort,
		Password:     raw.RedisPassword,
		PoolSize:     raw.RedisPoolSize,
		MinIdleConns: raw.RedisMinIdle,
	}
}

func buildRateLimit(raw *rawConfig) RateLimit {
	return RateLimit{
		SubmitFlag:         raw.RateLimitSubmitFlag,
		SubmitFlagDuration: time.Duration(raw.RateLimitSubmitFlagDuration) * time.Minute,
	}
}

func buildResend(raw *rawConfig) Resend {
	apiKey := strings.TrimSpace(raw.ResendAPIKey)

	return Resend{
		APIKey:      apiKey,
		FromEmail:   raw.ResendFromEmail,
		FromName:    raw.ResendFromName,
		Enabled:     raw.ResendEnabled && isUsableResendAPIKey(apiKey),
		VerifyTTL:   time.Duration(raw.ResendVerifyTTLHrs) * time.Hour,
		ResetTTL:    time.Duration(raw.ResendResetTTLHrs) * time.Hour,
		FrontendURL: raw.FrontendURL,
	}
}

func buildStorage(raw *rawConfig) Storage {
	return Storage{
		Provider:         raw.StorageProvider,
		LocalPath:        raw.StorageLocalPath,
		S3Endpoint:       raw.S3Endpoint,
		S3PublicEndpoint: raw.S3PublicEndpoint,
		S3AccessKey:      raw.S3AccessKey,
		S3SecretKey:      raw.S3SecretKey,
		S3Bucket:         raw.S3Bucket,
		S3Region:         raw.S3Region,
		S3UseSSL:         raw.S3UseSSL,
		PresignedExpiry:  time.Duration(raw.StoragePresignedExpiryMin) * time.Minute,
	}
}

func validateStorageConfig(storage Storage) error {
	if storage.Provider == "s3" && (storage.S3Endpoint == "" || storage.S3Bucket == "") {
		return fmt.Errorf("config: S3_ENDPOINT and S3_BUCKET are required when STORAGE_PROVIDER=s3")
	}

	return nil
}

func buildCompetition(raw *rawConfig) Competition {
	return Competition{
		Mode:            raw.CompetitionMode,
		AllowTeamSwitch: raw.AllowTeamSwitch,
		MinTeamSize:     raw.MinTeamSize,
		MaxTeamSize:     raw.MaxTeamSize,
	}
}

func warnCompetitionConfig(competition Competition, l logkit.Logger) {
	if competition.Mode == string(domain.ModeSoloOnly) && competition.MinTeamSize > 1 {
		l.Warn("Config: COMPETITION_MODE=solo_only with MIN_TEAM_SIZE>1 misconfigures flag submit for solo; set MIN_TEAM_SIZE=1 or change mode",
			logkit.Fields{"min_team_size": competition.MinTeamSize})
	}
}

func buildOAuth(raw *rawConfig) OAuth {
	return OAuth{
		StateSecret: raw.OAuthStateSecret,
		GitHub: OAuthProvider{
			ClientID:     raw.OAuthGitHubClientID,
			ClientSecret: raw.OAuthGitHubClientSecret,
			RedirectURL:  raw.OAuthGitHubRedirectURL,
		},
		Google: OAuthProvider{
			ClientID:     raw.OAuthGoogleClientID,
			ClientSecret: raw.OAuthGoogleClientSecret,
			RedirectURL:  raw.OAuthGoogleRedirectURL,
		},
	}
}
