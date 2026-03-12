package config

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	pkgjwt "github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
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
		ChiMode           string
		LogLevel          string
		FlagEncryptionKey string
		VerifyEmails      bool
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
		AccessSecret  string
		RefreshSecret string
		AccessKeys    []JWTKey
		RefreshKeys   []JWTKey
		AccessTTL     time.Duration
		RefreshTTL    time.Duration
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

type rawConfig struct {
	AppName, AppVersion, ChiMode, LogLevel, FlagEncryptionKey                              string
	VerifyEmails                                                                           bool
	BackendPort, APIBaseURL, MigrationsPath                                                string
	CORSOrigins                                                                            []string
	TrustedProxyCIDRs, MetricsAllowedIPs                                                   []string
	ShutdownTimeoutSec                                                                     int
	PostgresHost, PostgresPort, PostgresUser, PostgresPassword, PostgresDB                 string
	RedisHost, RedisPort, RedisPassword                                                    string
	RedisPoolSize, RedisMinIdle                                                            int
	JWTAccessSecret, JWTRefreshSecret                                                      string
	JWTAccessTTLMin, JWTRefreshTTLHrs                                                      int
	ResendAPIKey, S3AccessKey, S3SecretKey                                                 string
	AdminUsername, AdminEmail, AdminPassword                                               string
	RateLimitSubmitFlag, RateLimitSubmitFlagDuration                                       int
	ResendFromEmail, ResendFromName, FrontendURL                                           string
	ResendEnabled                                                                          bool
	ResendVerifyTTLHrs, ResendResetTTLHrs                                                  int
	StorageProvider, StorageLocalPath                                                      string
	S3Endpoint, S3PublicEndpoint, S3Bucket, S3Region                                       string
	S3UseSSL                                                                               bool
	StoragePresignedExpiryMin                                                              int
	CompetitionMode                                                                        string
	AllowTeamSwitch                                                                        bool
	MinTeamSize, MaxTeamSize                                                               int
	OAuthStateSecret, OAuthGitHubClientID, OAuthGitHubClientSecret, OAuthGitHubRedirectURL string
	OAuthGoogleClientID, OAuthGoogleClientSecret, OAuthGoogleRedirectURL                   string
	DBSSLMode                                                                              string
}

func loadFromEnv() *rawConfig {
	envPaths := []string{".env", "../.env", "../../.env", "/app/.env"}
	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("[config] .env file loaded from %s", path)
			envLoaded = true
			break
		}
	}
	if !envLoaded {
		log.Println("[config] .env file not found, using environment variables (production mode)")
	}

	raw := &rawConfig{}
	raw.AppName = getEnv("APP_NAME", "AstroCTFb")
	raw.AppVersion = getEnv("APP_VERSION", "1.0.0")
	raw.ChiMode = getEnv("CHI_MODE", "production")
	raw.LogLevel = getEnv("LOG_LEVEL", "info")
	raw.FlagEncryptionKey = getEnv("FLAG_ENCRYPTION_KEY", "")
	raw.VerifyEmails = getEnvBool("VERIFY_EMAILS", false)
	raw.BackendPort = getEnv("BACKEND_PORT", "8080")
	raw.APIBaseURL = getEnv("API_BASE_URL", "http://localhost:8080")
	raw.MigrationsPath = getEnv("MIGRATIONS_PATH", "migrations")
	raw.CORSOrigins = parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173,http://localhost:5000"))
	raw.TrustedProxyCIDRs = parseTrustedProxyCIDRs(getEnv("TRUSTED_PROXY_CIDRS", ""))
	raw.MetricsAllowedIPs = parseCommaSeparated(getEnv("METRICS_ALLOWED_IPS", ""))
	raw.ShutdownTimeoutSec = getEnvInt("HTTP_SHUTDOWN_TIMEOUT", 15)
	if raw.ShutdownTimeoutSec < 1 {
		raw.ShutdownTimeoutSec = 15
	}
	raw.PostgresHost = getEnv("POSTGRES_HOST", "postgres")
	raw.PostgresPort = getEnv("POSTGRES_PORT", "5432")
	raw.PostgresUser = getEnv("POSTGRES_USER", "")
	raw.PostgresPassword = getEnv("POSTGRES_PASSWORD", "")
	raw.PostgresDB = getEnv("POSTGRES_DB", "")
	raw.RedisHost = getEnv("REDIS_HOST", "redis")
	raw.RedisPort = getEnv("REDIS_PORT", "6379")
	raw.RedisPassword = getEnv("REDIS_PASSWORD", "")
	raw.RedisPoolSize = getEnvInt("REDIS_POOL_SIZE", 50)
	raw.RedisMinIdle = getEnvInt("REDIS_MIN_IDLE", 10)
	raw.JWTAccessSecret = getEnv("JWT_ACCESS_SECRET", "")
	raw.JWTRefreshSecret = getEnv("JWT_REFRESH_SECRET", "")
	raw.JWTAccessTTLMin = getEnvInt("JWT_ACCESS_TTL_MINUTES", 15)
	raw.JWTRefreshTTLHrs = getEnvInt("JWT_REFRESH_TTL_HOURS", 72)
	raw.ResendAPIKey = getEnv("RESEND_API_KEY", "")
	raw.S3AccessKey = getEnv("STORAGE_S3_ACCESS_KEY", "")
	raw.S3SecretKey = getEnv("STORAGE_S3_SECRET_KEY", "")
	raw.AdminUsername = getEnv("ADMIN_USERNAME", "")
	raw.AdminEmail = getEnv("ADMIN_EMAIL", "")
	raw.AdminPassword = getEnv("ADMIN_PASSWORD", "")
	raw.RateLimitSubmitFlag = getEnvInt("RATE_LIMIT_SUBMIT_FLAG", 10)
	raw.RateLimitSubmitFlagDuration = getEnvInt("RATE_LIMIT_SUBMIT_FLAG_DURATION", 1)
	raw.ResendFromEmail = getEnv("RESEND_FROM_EMAIL", "noreply@astroctfb.local")
	raw.ResendFromName = getEnv("RESEND_FROM_NAME", "AstroCTFb")
	raw.ResendEnabled = getEnvBool("RESEND_ENABLED", false)
	raw.ResendVerifyTTLHrs = getEnvInt("RESEND_VERIFY_TTL_HOURS", 24)
	raw.ResendResetTTLHrs = getEnvInt("RESEND_RESET_TTL_HOURS", 1)
	raw.FrontendURL = getEnv("FRONTEND_URL", "http://localhost:3000")
	raw.StorageProvider = getEnv("STORAGE_PROVIDER", "filesystem")
	raw.StorageLocalPath = getEnv("STORAGE_LOCAL_PATH", "./uploads")
	s3DefaultEndpoint, s3DefaultBucket := "urchin:9000", "tasks"
	if raw.StorageProvider == "s3" {
		s3DefaultEndpoint, s3DefaultBucket = "", ""
	}
	raw.S3Endpoint = getEnv("STORAGE_S3_ENDPOINT", s3DefaultEndpoint)
	raw.S3PublicEndpoint = getEnv("STORAGE_S3_PUBLIC_ENDPOINT", "")
	raw.S3Bucket = getEnv("STORAGE_S3_BUCKET", s3DefaultBucket)
	raw.S3Region = getEnv("STORAGE_S3_REGION", "us-east-1")
	raw.S3UseSSL = getEnvBool("STORAGE_S3_USE_SSL", false)
	raw.StoragePresignedExpiryMin = getEnvInt("STORAGE_PRESIGNED_EXPIRY_MINUTES", 60)
	raw.CompetitionMode = getEnv("COMPETITION_MODE", "flexible")
	raw.AllowTeamSwitch = getEnvBool("ALLOW_TEAM_SWITCH", true)
	raw.MinTeamSize = getEnvInt("MIN_TEAM_SIZE", 1)
	raw.MaxTeamSize = getEnvInt("MAX_TEAM_SIZE", 10)
	raw.OAuthStateSecret = getEnv("OAUTH_STATE_SECRET", "")
	raw.OAuthGitHubClientID = getEnv("OAUTH_GITHUB_CLIENT_ID", "")
	raw.OAuthGitHubClientSecret = getEnv("OAUTH_GITHUB_CLIENT_SECRET", "")
	raw.OAuthGitHubRedirectURL = getEnv("OAUTH_GITHUB_REDIRECT_URL", "")
	raw.OAuthGoogleClientID = getEnv("OAUTH_GOOGLE_CLIENT_ID", "")
	raw.OAuthGoogleClientSecret = getEnv("OAUTH_GOOGLE_CLIENT_SECRET", "")
	raw.OAuthGoogleRedirectURL = getEnv("OAUTH_GOOGLE_REDIRECT_URL", "")
	raw.DBSSLMode = getEnv("POSTGRES_SSL_MODE", "disable")
	return raw
}

func loadFromVault(ctx context.Context, raw *rawConfig, l logger.Logger) {
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
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/database", "database", "using env", apply(func(s map[string]any) {
		if u, ok := s[string(entity.RoleUser)].(string); ok && u != "" {
			raw.PostgresUser = u
		}
		if p, ok := s["password"].(string); ok && p != "" {
			raw.PostgresPassword = p
		}
		if db, ok := s["dbname"].(string); ok && db != "" {
			raw.PostgresDB = db
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/redis", "redis", "using env", apply(func(s map[string]any) {
		if p, ok := s["password"].(string); ok && p != "" {
			raw.RedisPassword = p
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/jwt", "jwt", "using env", apply(func(s map[string]any) {
		if access, ok := s["access_secret"].(string); ok && access != "" {
			raw.JWTAccessSecret = access
		}
		if refresh, ok := s["refresh_secret"].(string); ok && refresh != "" {
			raw.JWTRefreshSecret = refresh
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/resend", "Resend", "using env (or not configured)", apply(func(s map[string]any) {
		if k, ok := s["api_key"].(string); ok && k != "" {
			raw.ResendAPIKey = k
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/storage", "Storage", "(optional)", apply(func(s map[string]any) {
		if k, ok := s["access_key"].(string); ok && k != "" {
			raw.S3AccessKey = k
		}
		if sec, ok := s["secret_key"].(string); ok && sec != "" {
			raw.S3SecretKey = sec
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/app", "app", "using env", apply(func(s map[string]any) {
		if key, ok := s["flag_encryption_key"].(string); ok && key != "" {
			raw.FlagEncryptionKey = key
		}
	})))
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/admin", "admin", "using env (optional)", apply(func(s map[string]any) {
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
	g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/oauth", "OAuth", "using env (optional)", apply(func(s map[string]any) {
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

func validate(raw *rawConfig) error {
	if raw.PostgresUser == "" || raw.PostgresPassword == "" || raw.PostgresDB == "" {
		return fmt.Errorf("required database configuration is missing (env or vault)")
	}
	if raw.JWTAccessSecret == "" || raw.JWTRefreshSecret == "" {
		return fmt.Errorf("required jwt configuration is missing (env or vault)")
	}
	if len(raw.JWTAccessSecret) < pkgjwt.MinSecretLength {
		return fmt.Errorf("JWT_ACCESS_SECRET must be at least %d bytes, got %d", pkgjwt.MinSecretLength, len(raw.JWTAccessSecret))
	}
	if len(raw.JWTRefreshSecret) < pkgjwt.MinSecretLength {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least %d bytes, got %d", pkgjwt.MinSecretLength, len(raw.JWTRefreshSecret))
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
	if (raw.OAuthGitHubClientID != "" || raw.OAuthGoogleClientID != "") && raw.OAuthStateSecret == "" {
		return fmt.Errorf("OAUTH_STATE_SECRET is required when OAuth clients are configured")
	}
	if !entity.CompetitionMode(raw.CompetitionMode).IsValid() {
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
	return nil
}

func buildConfig(raw *rawConfig) (*Config, error) {
	jwtAccessKeys := []JWTKey{{Kid: "0", Secret: raw.JWTAccessSecret}}
	if s := getEnv("JWT_ACCESS_KEYS", ""); s != "" {
		var parsed []JWTKey
		if err := json.Unmarshal([]byte(s), &parsed); err != nil {
			return nil, fmt.Errorf("JWT_ACCESS_KEYS invalid JSON: %w", err)
		}
		if len(parsed) == 0 {
			return nil, fmt.Errorf("JWT_ACCESS_KEYS must contain at least one key")
		}
		for i, k := range parsed {
			if len(k.Secret) < pkgjwt.MinSecretLength {
				return nil, fmt.Errorf("JWT_ACCESS_KEYS[%d] secret must be at least %d bytes", i, pkgjwt.MinSecretLength)
			}
		}
		jwtAccessKeys = parsed
	}
	jwtRefreshKeys := []JWTKey{{Kid: "0", Secret: raw.JWTRefreshSecret}}
	if s := getEnv("JWT_REFRESH_KEYS", ""); s != "" {
		var parsed []JWTKey
		if err := json.Unmarshal([]byte(s), &parsed); err != nil {
			return nil, fmt.Errorf("JWT_REFRESH_KEYS invalid JSON: %w", err)
		}
		if len(parsed) == 0 {
			return nil, fmt.Errorf("JWT_REFRESH_KEYS must contain at least one key")
		}
		for i, k := range parsed {
			if len(k.Secret) < pkgjwt.MinSecretLength {
				return nil, fmt.Errorf("JWT_REFRESH_KEYS[%d] secret must be at least %d bytes", i, pkgjwt.MinSecretLength)
			}
		}
		jwtRefreshKeys = parsed
	}

	dbURL := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(raw.PostgresUser, raw.PostgresPassword),
		Host:     raw.PostgresHost + ":" + raw.PostgresPort,
		Path:     raw.PostgresDB,
		RawQuery: "sslmode=" + url.QueryEscape(raw.DBSSLMode),
	}).String()

	cfg := &Config{
		App: App{
			Name:              raw.AppName,
			Version:           raw.AppVersion,
			ChiMode:           raw.ChiMode,
			LogLevel:          raw.LogLevel,
			FlagEncryptionKey: raw.FlagEncryptionKey,
			VerifyEmails:      raw.VerifyEmails,
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
			MaxConns:       getEnvInt("POSTGRES_MAX_CONNS", 100),
			MinConns:       getEnvInt("POSTGRES_MIN_CONNS", 10),
		},
		JWT: JWT{
			AccessSecret:  raw.JWTAccessSecret,
			RefreshSecret: raw.JWTRefreshSecret,
			AccessKeys:    jwtAccessKeys,
			RefreshKeys:   jwtRefreshKeys,
			AccessTTL:     time.Duration(raw.JWTAccessTTLMin) * time.Minute,
			RefreshTTL:    time.Duration(raw.JWTRefreshTTLHrs) * time.Hour,
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
			APIKey:      raw.ResendAPIKey,
			FromEmail:   raw.ResendFromEmail,
			FromName:    raw.ResendFromName,
			Enabled:     raw.ResendEnabled,
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
	if cfg.Enabled && cfg.APIKey == "" {
		return nil, fmt.Errorf("config: RESEND_API_KEY is required when RESEND_ENABLED=true")
	}
	if cfg.Mode == string(entity.ModeSoloOnly) && cfg.MinTeamSize > 1 {
		log.Printf("WARNING: COMPETITION_MODE=solo_only with MIN_TEAM_SIZE=%d > 1 is a misconfiguration: "+
			"solo teams always have exactly 1 member, so all solo players will be blocked from submitting flags. "+
			"Set MIN_TEAM_SIZE=1 or switch to a team mode.", cfg.MinTeamSize)
	}
	if cfg.Mode == string(entity.ModeFlexible) && cfg.MinTeamSize > 1 {
		log.Printf("WARNING: COMPETITION_MODE=flexible with MIN_TEAM_SIZE=%d > 1: "+
			"MinTeamSize applies only to multi-member teams; solo teams are exempt.", cfg.MinTeamSize)
	}
	return cfg, nil
}

func New() (*Config, error) {
	raw := loadFromEnv()
	var lvl logger.Level
	switch raw.LogLevel {
	case "debug":
		lvl = logger.DebugLevel
	case "warn":
		lvl = logger.WarnLevel
	case "error":
		lvl = logger.ErrorLevel
	default:
		lvl = logger.InfoLevel
	}
	l := logger.New(&logger.Options{Level: lvl, Output: logger.ConsoleOutput})
	vaultCtx, vaultCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer vaultCancel()
	loadFromVault(vaultCtx, raw, l)
	if err := validate(raw); err != nil {
		return nil, err
	}
	return buildConfig(raw)
}
