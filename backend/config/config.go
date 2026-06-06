package config

import (
	"context"
	"fmt"
	"time"

	"github.com/wahrwelt-kit/go-logkit"
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

const (
	flagEncryptionKeyHexLen = 64
	vaultLoadTimeout        = 30 * time.Second
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

	vaultCtx, vaultCancel := context.WithTimeout(context.Background(), vaultLoadTimeout)
	defer vaultCancel()

	loadFromVault(vaultCtx, raw, l)

	if err := validate(raw); err != nil {
		return nil, err
	}

	return buildConfig(raw, l)
}
