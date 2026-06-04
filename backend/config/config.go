package config

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/vault"
)

type (
	Config struct {
		App         `yaml:"app"`
		Admin       `yaml:"admin"`
		HTTP        `yaml:"http"`
		DB          `yaml:"postgres"`
		JWT         `yaml:"jwt"`
		Redis       `yaml:"redis"`
		RateLimit   `yaml:"rate_limit"`
		Resend      `yaml:"resend"`
		Storage     `yaml:"storage"`
		Competition `yaml:"competition"`
		OAuth       `yaml:"oauth"`
	}

	App struct {
		Name              string
		Version           string
		StructuredLogger  bool
		SecureCookies     bool
		SetupToken        string
		LogLevel          string
		FlagEncryptionKey string
		VerifyEmails      bool
		DebugEnabled      bool
	}

	Admin struct {
		Username string
		Email    string
		Password string
	}

	HTTP struct {
		Port              string
		BaseURL           string
		CORSOrigins       []string
		TrustedProxyCIDRs []string
		MetricsAllowedIPs []string
		ShutdownTimeout   time.Duration
	}

	DB struct {
		URL            string
		MigrationsPath string
		MaxConns       int
		MinConns       int
	}

	JWT struct {
		AccessSecret   string
		RefreshSecret  string
		DownloadSecret string
		AccessKeys     []JWTKey
		RefreshKeys    []JWTKey
		AccessTTL      time.Duration
		RefreshTTL     time.Duration
		Issuer         string
	}

	JWTKey struct {
		Kid    string `json:"kid"`
		Secret string `json:"secret"`
	}

	Redis struct {
		Host         string
		Port         string
		Password     string
		PoolSize     int
		MinIdleConns int
	}

	RateLimit struct {
		SubmitFlag         int
		SubmitFlagDuration time.Duration
	}

	Resend struct {
		APIKey      string
		FromEmail   string
		FromName    string
		Enabled     bool
		VerifyTTL   time.Duration
		ResetTTL    time.Duration
		FrontendURL string
	}

	Storage struct {
		Provider         string
		LocalPath        string
		S3Endpoint       string
		S3PublicEndpoint string
		S3AccessKey      string
		S3SecretKey      string
		S3Bucket         string
		S3Region         string
		S3UseSSL         bool
		PresignedExpiry  time.Duration
	}

	Competition struct {
		Mode            string
		AllowTeamSwitch bool
		MinTeamSize     int
		MaxTeamSize     int
	}

	OAuth struct {
		StateSecret string
		GitHub      OAuthProvider
		Google      OAuthProvider
	}

	OAuthProvider struct {
		ClientID     string
		ClientSecret string
		RedirectURL  string
	}
)

func (p OAuthProvider) IsConfigured() bool {
	return p.ClientID != "" && p.ClientSecret != "" && p.RedirectURL != ""
}

type rawConfig struct {
	AppName                     string `env:"APP_NAME"                         env-default:"CTF Platform"`
	AppVersion                  string `env:"APP_VERSION"                      env-default:"1.0.0"`
	StructuredLogger            bool   `env:"STRUCTURED_LOGGER"                env-default:"true"`
	SecureCookies               bool   `env:"SECURE_COOKIES"                   env-default:"false"`
	SetupToken                  string `env:"SETUP_TOKEN"`
	DebugEnabled                bool   `env:"DEBUG_ENABLED"                    env-default:"false"`
	LogLevel                    string `env:"LOG_LEVEL"                        env-default:"info"`
	FlagEncryptionKey           string `env:"FLAG_ENCRYPTION_KEY"`
	VerifyEmails                bool   `env:"VERIFY_EMAILS"                    env-default:"false"`
	BackendPort                 string `env:"BACKEND_PORT"                     env-default:"8080"`
	APIBaseURL                  string `env:"API_BASE_URL"                     env-default:"http://localhost:8080"`
	MigrationsPath              string `env:"MIGRATIONS_PATH"                  env-default:"migrations"`
	CORSOriginsStr              string `env:"CORS_ORIGINS"                     env-default:"http://localhost:3000,http://localhost:5173,http://localhost:5000"`
	TrustedProxyCIDRsStr        string `env:"TRUSTED_PROXY_CIDRS"`
	MetricsAllowedIPsStr        string `env:"METRICS_ALLOWED_IPS"`
	ShutdownTimeoutSec          int    `env:"HTTP_SHUTDOWN_TIMEOUT"            env-default:"15"`
	PostgresHost                string `env:"POSTGRES_HOST"                    env-default:"postgres"`
	PostgresPort                string `env:"POSTGRES_PORT"                    env-default:"5432"`
	PostgresUser                string `env:"POSTGRES_USER"`
	PostgresPassword            string `env:"POSTGRES_PASSWORD"`
	PostgresDB                  string `env:"POSTGRES_DB"`
	PostgresMaxConns            int    `env:"POSTGRES_MAX_CONNS"               env-default:"100"`
	PostgresMinConns            int    `env:"POSTGRES_MIN_CONNS"               env-default:"10"`
	RedisHost                   string `env:"REDIS_HOST"                       env-default:"redis"`
	RedisPort                   string `env:"REDIS_PORT"                       env-default:"6379"`
	RedisPassword               string `env:"REDIS_PASSWORD"`
	RedisPoolSize               int    `env:"REDIS_POOL_SIZE"                  env-default:"50"`
	RedisMinIdle                int    `env:"REDIS_MIN_IDLE"                   env-default:"10"`
	JWTAccessSecret             string `env:"JWT_ACCESS_SECRET"`
	JWTRefreshSecret            string `env:"JWT_REFRESH_SECRET"`
	JWTDownloadSecret           string `env:"JWT_DOWNLOAD_SECRET"`
	JWTAccessKeysStr            string `env:"JWT_ACCESS_KEYS"                  env-default:""`
	JWTRefreshKeysStr           string `env:"JWT_REFRESH_KEYS"                 env-default:""`
	JWTAccessTTLMin             int    `env:"JWT_ACCESS_TTL_MINUTES"           env-default:"15"`
	JWTRefreshTTLHrs            int    `env:"JWT_REFRESH_TTL_HOURS"            env-default:"72"`
	JWTIssuer                   string `env:"JWT_ISSUER"                       env-default:"ctf-platform"`
	ResendAPIKey                string `env:"RESEND_API_KEY"`
	S3AccessKey                 string `env:"STORAGE_S3_ACCESS_KEY"`
	S3SecretKey                 string `env:"STORAGE_S3_SECRET_KEY"`
	AdminUsername               string `env:"ADMIN_USERNAME"`
	AdminEmail                  string `env:"ADMIN_EMAIL"`
	AdminPassword               string `env:"ADMIN_PASSWORD"`
	RateLimitSubmitFlag         int    `env:"RATE_LIMIT_SUBMIT_FLAG"           env-default:"10"`
	RateLimitSubmitFlagDuration int    `env:"RATE_LIMIT_SUBMIT_FLAG_DURATION"  env-default:"1"`
	ResendFromEmail             string `env:"RESEND_FROM_EMAIL"                env-default:"noreply@ctf-platform.local"`
	ResendFromName              string `env:"RESEND_FROM_NAME"                 env-default:"CTF Platform"`
	ResendEnabled               bool   `env:"RESEND_ENABLED"                   env-default:"false"`
	ResendVerifyTTLHrs          int    `env:"RESEND_VERIFY_TTL_HOURS"          env-default:"24"`
	ResendResetTTLHrs           int    `env:"RESEND_RESET_TTL_HOURS"           env-default:"1"`
	FrontendURL                 string `env:"FRONTEND_URL"                     env-default:"http://localhost:3000"`
	StorageProvider             string `env:"STORAGE_PROVIDER"                 env-default:"filesystem"`
	StorageLocalPath            string `env:"STORAGE_LOCAL_PATH"               env-default:"./uploads"`
	S3Endpoint                  string `env:"STORAGE_S3_ENDPOINT"              env-default:"urchin:9000"`
	S3PublicEndpoint            string `env:"STORAGE_S3_PUBLIC_ENDPOINT"`
	S3Bucket                    string `env:"STORAGE_S3_BUCKET"                env-default:"ctf"`
	S3Region                    string `env:"STORAGE_S3_REGION"                env-default:"us-east-1"`
	S3UseSSL                    bool   `env:"STORAGE_S3_USE_SSL"               env-default:"false"`
	StoragePresignedExpiryMin   int    `env:"STORAGE_PRESIGNED_EXPIRY_MINUTES" env-default:"60"`
	CompetitionMode             string `env:"COMPETITION_MODE"                 env-default:"flexible"`
	AllowTeamSwitch             bool   `env:"ALLOW_TEAM_SWITCH"                env-default:"true"`
	MinTeamSize                 int    `env:"MIN_TEAM_SIZE"                    env-default:"1"`
	MaxTeamSize                 int    `env:"MAX_TEAM_SIZE"                    env-default:"10"`
	OAuthStateSecret            string `env:"OAUTH_STATE_SECRET"`
	OAuthGitHubClientID         string `env:"OAUTH_GITHUB_CLIENT_ID"`
	OAuthGitHubClientSecret     string `env:"OAUTH_GITHUB_CLIENT_SECRET"`
	OAuthGitHubRedirectURL      string `env:"OAUTH_GITHUB_REDIRECT_URL"`
	OAuthGoogleClientID         string `env:"OAUTH_GOOGLE_CLIENT_ID"`
	OAuthGoogleClientSecret     string `env:"OAUTH_GOOGLE_CLIENT_SECRET"`
	OAuthGoogleRedirectURL      string `env:"OAUTH_GOOGLE_REDIRECT_URL"`
	DBSSLMode                   string `env:"POSTGRES_SSL_MODE"                env-default:"disable"`

	CORSOrigins       []string
	TrustedProxyCIDRs []string
	MetricsAllowedIPs []string
}

// loadFromEnv probes .env, ../.env, and /app/.env in order, loading the first
// file found via godotenv. It then reads all env-tagged fields into rawConfig
// via cleanenv (missing optional fields use struct defaults). Post-load it
// splits CORS_ORIGINS and TRUSTED_PROXY_CIDRS into slices, and clears the
// placeholder S3 endpoint/bucket when the provider is s3 but defaults are
// still set (prevents accidental connection to the dev SeaweedFS instance).
func loadFromEnv(l logkit.Logger) *rawConfig {
	envPaths := []string{".env", "../.env", "/app/.env"}
	for _, path := range envPaths {
		err := godotenv.Load(path)
		if err == nil {
			l.Info("Config: .env file loaded", logkit.Fields{"path": path})

			break
		}
	}

	raw := &rawConfig{}

	err := cleanenv.ReadEnv(raw)
	if err != nil {
		l.WithError(err).Warn("Config: cleanenv.ReadEnv (using defaults where set)")
	}

	raw.CORSOrigins = parseCORSOrigins(raw.CORSOriginsStr)
	raw.TrustedProxyCIDRs = parseTrustedProxyCIDRs(raw.TrustedProxyCIDRsStr, l)

	raw.MetricsAllowedIPs = parseCommaSeparated(raw.MetricsAllowedIPsStr)
	if raw.ShutdownTimeoutSec < 1 {
		raw.ShutdownTimeoutSec = 15
	}

	if raw.StorageProvider == "s3" && raw.S3Endpoint == "urchin:9000" {
		raw.S3Endpoint = ""
		raw.S3Bucket = ""
	}

	return raw
}

// loadFromVault fetches secrets from Vault in parallel via errgroup (8 goroutines,
// one per secret path: database, redis, jwt, resend, storage, app, admin, oauth).
// Each goroutine uses vaultFetch, which silently logs a warning and returns nil on
// missing secrets so that a partial Vault setup does not block startup. A mutex
// protects concurrent writes into raw. The entire fetch is bounded by the caller's
// context (typically 30 s). If VAULT_ADDR or VAULT_TOKEN are absent the function
// returns immediately, leaving raw unchanged.
func loadFromVault(ctx context.Context, raw *rawConfig, l logkit.Logger) {
	vaultAddr := os.Getenv("VAULT_ADDR")

	vaultToken := os.Getenv("VAULT_TOKEN")
	if vaultAddr == "" || vaultToken == "" {
		return
	}

	l.Info("Config: attempting to fetch secrets from Vault")

	vaultClient, err := vault.New(vaultAddr, vaultToken)
	if err != nil {
		l.WithError(err).Error("Config: failed to initialize vault client")

		return
	}

	var mu sync.Mutex

	apply := func(fn func(map[string]any)) func(map[string]any) {
		return func(s map[string]any) {
			mu.Lock()
			defer mu.Unlock()

			fn(s)
		}
	}
	g, gCtx := errgroup.WithContext(ctx)
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/database", "database", "using env", apply(func(s map[string]any) {
		if u, ok := s[string(domain.RoleUser)].(string); ok && u != "" {
			raw.PostgresUser = u
		}

		if p, ok := s["password"].(string); ok && p != "" {
			raw.PostgresPassword = p
		}

		if db, ok := s["dbname"].(string); ok && db != "" {
			raw.PostgresDB = db
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/redis", "redis", "using env", apply(func(s map[string]any) {
		if p, ok := s["password"].(string); ok && p != "" {
			raw.RedisPassword = p
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/jwt", "jwt", "using env", apply(func(s map[string]any) {
		if access, ok := s["access_secret"].(string); ok && access != "" {
			raw.JWTAccessSecret = access
		}

		if refresh, ok := s["refresh_secret"].(string); ok && refresh != "" {
			raw.JWTRefreshSecret = refresh
		}

		if download, ok := s["download_secret"].(string); ok && download != "" {
			raw.JWTDownloadSecret = download
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/resend", "Resend", "using env (or not configured)", apply(func(s map[string]any) {
		if k, ok := s["api_key"].(string); ok && k != "" {
			raw.ResendAPIKey = k
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/storage", "Storage", "(optional)", apply(func(s map[string]any) {
		if k, ok := s["access_key"].(string); ok && k != "" {
			raw.S3AccessKey = k
		}

		if sec, ok := s["secret_key"].(string); ok && sec != "" {
			raw.S3SecretKey = sec
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/app", "app", "using env", apply(func(s map[string]any) {
		if key, ok := s["flag_encryption_key"].(string); ok && key != "" {
			raw.FlagEncryptionKey = key
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/admin", "admin", "using env (optional)", apply(func(s map[string]any) {
		if u, ok := s["username"].(string); ok && u != "" {
			raw.AdminUsername = u
		}

		if e, ok := s["email"].(string); ok && e != "" {
			raw.AdminEmail = e
		}

		if p, ok := s["password"].(string); ok && p != "" {
			raw.AdminPassword = p
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "ctf-platform/oauth", "OAuth", "using env (optional)", apply(func(s map[string]any) {
		if v, ok := s["state_secret"].(string); ok && v != "" {
			raw.OAuthStateSecret = v
		}

		if v, ok := s["github_client_id"].(string); ok && v != "" {
			raw.OAuthGitHubClientID = v
		}

		if v, ok := s["github_client_secret"].(string); ok && v != "" {
			raw.OAuthGitHubClientSecret = v
		}

		if v, ok := s["google_client_id"].(string); ok && v != "" {
			raw.OAuthGoogleClientID = v
		}

		if v, ok := s["google_client_secret"].(string); ok && v != "" {
			raw.OAuthGoogleClientSecret = v
		}
	})))

	if err := g.Wait(); err != nil {
		l.WithError(err).Warn("Config: vault goroutine error")
	}
}

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

	if len(raw.FlagEncryptionKey) != 64 {
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

// buildConfig assembles the final Config from a validated rawConfig. Key steps:
// JWT key rotation arrays (JWT_ACCESS_KEYS / JWT_REFRESH_KEYS) are parsed from
// JSON if present, otherwise the single primary secret is wrapped as kid="0".
// If JWT_DOWNLOAD_SECRET is empty, it is derived via HMAC-SHA256 over the access
// secret with the fixed label "download-url-signing". The Postgres DSN is
// constructed programmatically to avoid injection via url.URL. SecureCookies is
// forced true when API_BASE_URL starts with https://. Post-assembly warnings are
// emitted for known configuration anti-patterns (solo_only with MinTeamSize>1,
// flexible with MinTeamSize>1).
func buildConfig(raw *rawConfig, l logkit.Logger) (*Config, error) {
	jwtAccessKeys := []JWTKey{{Kid: "0", Secret: raw.JWTAccessSecret}}
	if s := raw.JWTAccessKeysStr; s != "" {
		var parsed []JWTKey

		err := json.Unmarshal([]byte(s), &parsed)
		if err != nil {
			return nil, fmt.Errorf("JWT_ACCESS_KEYS invalid JSON: %w", err)
		}

		if len(parsed) == 0 {
			return nil, fmt.Errorf("JWT_ACCESS_KEYS must contain at least one key")
		}

		for i, k := range parsed {
			if len(k.Secret) < jwtkit.MinSecretLength {
				return nil, fmt.Errorf("JWT_ACCESS_KEYS[%d] secret must be at least %d bytes", i, jwtkit.MinSecretLength)
			}
		}

		jwtAccessKeys = parsed
	}

	jwtRefreshKeys := []JWTKey{{Kid: "0", Secret: raw.JWTRefreshSecret}}
	if s := raw.JWTRefreshKeysStr; s != "" {
		var parsed []JWTKey

		err := json.Unmarshal([]byte(s), &parsed)
		if err != nil {
			return nil, fmt.Errorf("JWT_REFRESH_KEYS invalid JSON: %w", err)
		}

		if len(parsed) == 0 {
			return nil, fmt.Errorf("JWT_REFRESH_KEYS must contain at least one key")
		}

		for i, k := range parsed {
			if len(k.Secret) < jwtkit.MinSecretLength {
				return nil, fmt.Errorf("JWT_REFRESH_KEYS[%d] secret must be at least %d bytes", i, jwtkit.MinSecretLength)
			}
		}

		jwtRefreshKeys = parsed
	}

	downloadSecret := raw.JWTDownloadSecret
	if downloadSecret == "" {
		h := hmac.New(sha256.New, []byte(raw.JWTAccessSecret))
		h.Write([]byte("download-url-signing")) //nolint:revive // hash.Write never returns error
		downloadSecret = hex.EncodeToString(h.Sum(nil))
	}

	dbURL := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(raw.PostgresUser, raw.PostgresPassword),
		Host:     raw.PostgresHost + ":" + raw.PostgresPort,
		Path:     raw.PostgresDB,
		RawQuery: "sslmode=" + url.QueryEscape(raw.DBSSLMode),
	}).String()

	resendAPIKey := strings.TrimSpace(raw.ResendAPIKey)
	resendEnabled := raw.ResendEnabled && isUsableResendAPIKey(resendAPIKey)

	cfg := &Config{
		App: App{
			Name:              raw.AppName,
			Version:           raw.AppVersion,
			StructuredLogger:  raw.StructuredLogger,
			SecureCookies:     raw.SecureCookies || strings.HasPrefix(raw.APIBaseURL, "https://"),
			SetupToken:        raw.SetupToken,
			LogLevel:          raw.LogLevel,
			FlagEncryptionKey: raw.FlagEncryptionKey,
			VerifyEmails:      raw.VerifyEmails && resendEnabled,
			DebugEnabled:      raw.DebugEnabled,
		},
		Admin: Admin{
			Username: raw.AdminUsername,
			Email:    raw.AdminEmail,
			Password: raw.AdminPassword,
		},
		HTTP: HTTP{
			Port:              raw.BackendPort,
			BaseURL:           raw.APIBaseURL,
			CORSOrigins:       raw.CORSOrigins,
			TrustedProxyCIDRs: raw.TrustedProxyCIDRs,
			MetricsAllowedIPs: raw.MetricsAllowedIPs,
			ShutdownTimeout:   time.Duration(raw.ShutdownTimeoutSec) * time.Second,
		},
		DB: DB{
			URL:            dbURL,
			MigrationsPath: raw.MigrationsPath,
			MaxConns:       raw.PostgresMaxConns,
			MinConns:       raw.PostgresMinConns,
		},
		JWT: JWT{
			AccessSecret:   raw.JWTAccessSecret,
			RefreshSecret:  raw.JWTRefreshSecret,
			DownloadSecret: downloadSecret,
			AccessKeys:     jwtAccessKeys,
			RefreshKeys:    jwtRefreshKeys,
			AccessTTL:      time.Duration(raw.JWTAccessTTLMin) * time.Minute,
			RefreshTTL:     time.Duration(raw.JWTRefreshTTLHrs) * time.Hour,
			Issuer:         raw.JWTIssuer,
		},
		Redis: Redis{
			Host:         raw.RedisHost,
			Port:         raw.RedisPort,
			Password:     raw.RedisPassword,
			PoolSize:     raw.RedisPoolSize,
			MinIdleConns: raw.RedisMinIdle,
		},
		RateLimit: RateLimit{
			SubmitFlag:         raw.RateLimitSubmitFlag,
			SubmitFlagDuration: time.Duration(raw.RateLimitSubmitFlagDuration) * time.Minute,
		},
		Resend: Resend{
			APIKey:      resendAPIKey,
			FromEmail:   raw.ResendFromEmail,
			FromName:    raw.ResendFromName,
			Enabled:     resendEnabled,
			VerifyTTL:   time.Duration(raw.ResendVerifyTTLHrs) * time.Hour,
			ResetTTL:    time.Duration(raw.ResendResetTTLHrs) * time.Hour,
			FrontendURL: raw.FrontendURL,
		},
		Storage: Storage{
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
		},
		Competition: Competition{
			Mode:            raw.CompetitionMode,
			AllowTeamSwitch: raw.AllowTeamSwitch,
			MinTeamSize:     raw.MinTeamSize,
			MaxTeamSize:     raw.MaxTeamSize,
		},
		OAuth: OAuth{
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
		},
	}
	if cfg.Provider == "s3" {
		if cfg.S3Endpoint == "" || cfg.S3Bucket == "" {
			return nil, fmt.Errorf("config: S3_ENDPOINT and S3_BUCKET are required when STORAGE_PROVIDER=s3")
		}
	}

	if cfg.Mode == string(domain.ModeSoloOnly) && cfg.MinTeamSize > 1 {
		l.Warn("Config: COMPETITION_MODE=solo_only with MIN_TEAM_SIZE>1 misconfigures flag submit for solo; set MIN_TEAM_SIZE=1 or change mode",
			logkit.Fields{"min_team_size": cfg.MinTeamSize})
	}

	if cfg.Mode == string(domain.ModeFlexible) && cfg.MinTeamSize > 1 {
		l.Warn("Config: COMPETITION_MODE=flexible with MIN_TEAM_SIZE>1; MinTeamSize applies only to multi-member teams",
			logkit.Fields{"min_team_size": cfg.MinTeamSize})
	}

	return cfg, nil
}

// New builds a Config by running the full configuration pipeline:
// bootstrap logger -> loadFromEnv (multi-path .env probing + cleanenv) ->
// rebuild logger at the resolved log level -> loadFromVault (parallel Vault
// fetch with 30 s deadline) -> validate -> buildConfig.
// Any step failure is returned as a wrapped error; the caller (cmd/app/main.go)
// treats a non-nil error as a fatal startup failure.
func New() (*Config, error) {
	bootL, err := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		return nil, fmt.Errorf("config: bootstrap logger: %w", err)
	}

	raw := loadFromEnv(bootL)

	var lvl logkit.Level

	switch raw.LogLevel {
	case "debug":
		lvl = logkit.DebugLevel
	case "warn":
		lvl = logkit.WarnLevel
	case "error":
		lvl = logkit.ErrorLevel
	default:
		lvl = logkit.InfoLevel
	}

	l, err := logkit.New(logkit.WithLevel(lvl), logkit.WithOutput(logkit.ConsoleOutput))
	if err != nil {
		return nil, fmt.Errorf("config: create logger: %w", err)
	}

	vaultCtx, vaultCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer vaultCancel()

	loadFromVault(vaultCtx, raw, l)

	if err := validate(raw); err != nil {
		return nil, err
	}

	return buildConfig(raw, l)
}
