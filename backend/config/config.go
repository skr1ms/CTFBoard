package config

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	pkgjwt "github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/vault"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
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
		AccessTTL     time.Duration
		RefreshTTL    time.Duration
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

//nolint:gocognit,gocyclo,funlen
func New() (*Config, error) {
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

	// Initialize ALL variables from Environment first
	appName := getEnv("APP_NAME", "AstroCTFb")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	chiMode := getEnv("CHI_MODE", "production")
	logLevel := getEnv("LOG_LEVEL", "info")
	flagEncryptionKey := getEnv("FLAG_ENCRYPTION_KEY", "")
	verifyEmails := getEnvBool("VERIFY_EMAILS", false)
	backendPort := getEnv("BACKEND_PORT", "8080")
	apiBaseURL := getEnv("API_BASE_URL", "http://localhost:8080")
	migrationsPath := getEnv("MIGRATIONS_PATH", "migrations")
	corsOrigins := parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173,http://localhost:5000"))
	trustedProxyCIDRs := parseTrustedProxyCIDRs(getEnv("TRUSTED_PROXY_CIDRS", ""))
	metricsAllowedIPs := parseCommaSeparated(getEnv("METRICS_ALLOWED_IPS", ""))
	shutdownTimeoutSec := getEnvInt("HTTP_SHUTDOWN_TIMEOUT", 15)
	if shutdownTimeoutSec < 1 {
		shutdownTimeoutSec = 15
	}
	shutdownTimeout := time.Duration(shutdownTimeoutSec) * time.Second

	postgresHost := getEnv("POSTGRES_HOST", "postgres")
	postgresPort := getEnv("POSTGRES_PORT", "5432")
	postgresUser := getEnv("POSTGRES_USER", "")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "")
	postgresDB := getEnv("POSTGRES_DB", "")

	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	redisPoolSize := getEnvInt("REDIS_POOL_SIZE", 50)
	redisMinIdle := getEnvInt("REDIS_MIN_IDLE", 10)

	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	jwtAccessTTLMin := getEnvInt("JWT_ACCESS_TTL_MINUTES", 15)
	jwtRefreshTTLHrs := getEnvInt("JWT_REFRESH_TTL_HOURS", 72)

	resendAPIKey := getEnv("RESEND_API_KEY", "")
	s3AccessKey := getEnv("STORAGE_S3_ACCESS_KEY", "")
	s3SecretKey := getEnv("STORAGE_S3_SECRET_KEY", "")

	adminUsername := getEnv("ADMIN_USERNAME", "")
	adminEmail := getEnv("ADMIN_EMAIL", "")
	adminPassword := getEnv("ADMIN_PASSWORD", "")

	rateLimitSubmitFlag := getEnvInt("RATE_LIMIT_SUBMIT_FLAG", 10)
	rateLimitSubmitFlagDuration := time.Duration(getEnvInt("RATE_LIMIT_SUBMIT_FLAG_DURATION", 1)) * time.Minute

	resendFromEmail := getEnv("RESEND_FROM_EMAIL", "noreply@astroctfb.local")
	resendFromName := getEnv("RESEND_FROM_NAME", "AstroCTFb")
	resendEnabled := getEnvBool("RESEND_ENABLED", false)
	resendVerifyTTL := time.Duration(getEnvInt("RESEND_VERIFY_TTL_HOURS", 24)) * time.Hour
	resendResetTTL := time.Duration(getEnvInt("RESEND_RESET_TTL_HOURS", 1)) * time.Hour
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")

	storageProvider := getEnv("STORAGE_PROVIDER", "filesystem")
	storageLocalPath := getEnv("STORAGE_LOCAL_PATH", "./uploads")
	s3DefaultEndpoint := "urchin:9000"
	s3DefaultBucket := "tasks"
	if storageProvider == "s3" {
		s3DefaultEndpoint = ""
		s3DefaultBucket = ""
	}
	storageS3Endpoint := getEnv("STORAGE_S3_ENDPOINT", s3DefaultEndpoint)
	storageS3PublicEndpoint := getEnv("STORAGE_S3_PUBLIC_ENDPOINT", "")
	storageS3Bucket := getEnv("STORAGE_S3_BUCKET", s3DefaultBucket)
	storageS3Region := getEnv("STORAGE_S3_REGION", "us-east-1")
	storageS3UseSSL := getEnvBool("STORAGE_S3_USE_SSL", false)
	storagePresignedExpiry := time.Duration(getEnvInt("STORAGE_PRESIGNED_EXPIRY_MINUTES", 60)) * time.Minute

	competitionMode := getEnv("COMPETITION_MODE", "flexible")
	allowTeamSwitch := getEnvBool("ALLOW_TEAM_SWITCH", true)
	minTeamSize := getEnvInt("MIN_TEAM_SIZE", 1)
	maxTeamSize := getEnvInt("MAX_TEAM_SIZE", 10)

	oauthStateSecret := getEnv("OAUTH_STATE_SECRET", "")
	oauthGitHubClientID := getEnv("OAUTH_GITHUB_CLIENT_ID", "")
	oauthGitHubClientSecret := getEnv("OAUTH_GITHUB_CLIENT_SECRET", "")
	oauthGitHubRedirectURL := getEnv("OAUTH_GITHUB_REDIRECT_URL", "")
	oauthGoogleClientID := getEnv("OAUTH_GOOGLE_CLIENT_ID", "")
	oauthGoogleClientSecret := getEnv("OAUTH_GOOGLE_CLIENT_SECRET", "")
	oauthGoogleRedirectURL := getEnv("OAUTH_GOOGLE_REDIRECT_URL", "")

	var lvl logger.Level
	switch logLevel {
	case "debug":
		lvl = logger.DebugLevel
	case "warn":
		lvl = logger.WarnLevel
	case "error":
		lvl = logger.ErrorLevel
	default:
		lvl = logger.InfoLevel
	}

	l := logger.New(&logger.Options{
		Level:  lvl,
		Output: logger.ConsoleOutput,
	})

	// Try to fetch secrets from Vault and OVERRIDE if successful
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if vaultAddr != "" && vaultToken != "" {
		l.Info("Config: attempting to fetch secrets from Vault")
		vaultClient, err := vault.New(vaultAddr, vaultToken)
		if err == nil {
			g, gCtx := errgroup.WithContext(context.Background())

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/database", "database", "using env", func(s map[string]any) {
				if u, ok := s[entity.RoleUser].(string); ok && u != "" {
					postgresUser = u
				}
				if p, ok := s["password"].(string); ok && p != "" {
					postgresPassword = p
				}
				if db, ok := s["dbname"].(string); ok && db != "" {
					postgresDB = db
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/redis", "redis", "using env", func(s map[string]any) {
				if p, ok := s["password"].(string); ok && p != "" {
					redisPassword = p
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/jwt", "jwt", "using env", func(s map[string]any) {
				if access, ok := s["access_secret"].(string); ok && access != "" {
					jwtAccessSecret = access
				}
				if refresh, ok := s["refresh_secret"].(string); ok && refresh != "" {
					jwtRefreshSecret = refresh
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/resend", "Resend", "using env (or not configured)", func(s map[string]any) {
				if k, ok := s["api_key"].(string); ok && k != "" {
					resendAPIKey = k
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/storage", "Storage", "(optional)", func(s map[string]any) {
				if k, ok := s["access_key"].(string); ok && k != "" {
					s3AccessKey = k
				}
				if sec, ok := s["secret_key"].(string); ok && sec != "" {
					s3SecretKey = sec
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/app", "app", "using env", func(s map[string]any) {
				if key, ok := s["flag_encryption_key"].(string); ok && key != "" {
					flagEncryptionKey = key
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/admin", "admin", "using env (optional)", func(s map[string]any) {
				if u, ok := s["username"].(string); ok && u != "" {
					adminUsername = u
				}
				if e, ok := s["email"].(string); ok && e != "" {
					adminEmail = e
				}
				if p, ok := s["password"].(string); ok && p != "" {
					adminPassword = p
				}
			}))

			g.Go(vaultFetch(gCtx, vaultClient, l, "astroctfb/oauth", "OAuth", "using env (optional)", func(s map[string]any) {
				if v, ok := s["state_secret"].(string); ok && v != "" {
					oauthStateSecret = v
				}
				if v, ok := s["github_client_id"].(string); ok && v != "" {
					oauthGitHubClientID = v
				}
				if v, ok := s["github_client_secret"].(string); ok && v != "" {
					oauthGitHubClientSecret = v
				}
				if v, ok := s["google_client_id"].(string); ok && v != "" {
					oauthGoogleClientID = v
				}
				if v, ok := s["google_client_secret"].(string); ok && v != "" {
					oauthGoogleClientSecret = v
				}
			}))

			if err := g.Wait(); err != nil {
				l.WithError(err).Warn("Config: vault goroutine error")
			}
		} else {
			l.WithError(err).Error("Config: failed to initialize vault client")
		}
	}

	// Final Validation
	if postgresUser == "" || postgresPassword == "" || postgresDB == "" {
		return nil, fmt.Errorf("required database configuration is missing (env or vault)")
	}
	if jwtAccessSecret == "" || jwtRefreshSecret == "" {
		return nil, fmt.Errorf("required jwt configuration is missing (env or vault)")
	}
	if len(jwtAccessSecret) < pkgjwt.MinSecretLength {
		return nil, fmt.Errorf("JWT_ACCESS_SECRET must be at least %d bytes, got %d", pkgjwt.MinSecretLength, len(jwtAccessSecret))
	}
	if len(jwtRefreshSecret) < pkgjwt.MinSecretLength {
		return nil, fmt.Errorf("JWT_REFRESH_SECRET must be at least %d bytes, got %d", pkgjwt.MinSecretLength, len(jwtRefreshSecret))
	}
	if redisPassword == "" {
		return nil, fmt.Errorf("required redis configuration is missing (env or vault)")
	}
	if flagEncryptionKey == "" {
		return nil, fmt.Errorf("required flag encryption key is missing (env or vault) - needed for regex challenges")
	}
	if len(flagEncryptionKey) != 64 {
		return nil, fmt.Errorf("FLAG_ENCRYPTION_KEY must be exactly 64 hex characters (32 bytes for AES-256), got %d", len(flagEncryptionKey))
	}
	if (oauthGitHubClientID != "" || oauthGoogleClientID != "") && oauthStateSecret == "" {
		return nil, fmt.Errorf("OAUTH_STATE_SECRET is required when OAuth clients are configured")
	}
	if !entity.CompetitionMode(competitionMode).IsValid() {
		return nil, fmt.Errorf("invalid COMPETITION_MODE %q: must be solo_only, teams_only, or flexible", competitionMode)
	}
	if minTeamSize < 1 || maxTeamSize < minTeamSize {
		return nil, fmt.Errorf("invalid team size range: MIN_TEAM_SIZE=%d must be >= 1 and <= MAX_TEAM_SIZE=%d", minTeamSize, maxTeamSize)
	}
	switch storageProvider {
	case "filesystem", "s3":
	default:
		return nil, fmt.Errorf("invalid STORAGE_PROVIDER %q: must be filesystem or s3", storageProvider)
	}
	if rateLimitSubmitFlag <= 0 {
		return nil, fmt.Errorf("RATE_LIMIT_SUBMIT_FLAG must be a positive integer, got %d", rateLimitSubmitFlag)
	}

	dbSSLMode := getEnv("POSTGRES_SSL_MODE", "disable")
	dbURL := (&url.URL{
		Scheme:   "postgres",
		User:     url.UserPassword(postgresUser, postgresPassword),
		Host:     postgresHost + ":" + postgresPort,
		Path:     postgresDB,
		RawQuery: "sslmode=" + url.QueryEscape(dbSSLMode),
	}).String()

	cfg := &Config{
		App: App{
			Name:              appName,
			Version:           appVersion,
			ChiMode:           chiMode,
			LogLevel:          logLevel,
			FlagEncryptionKey: flagEncryptionKey,
			VerifyEmails:      verifyEmails,
		},
		Admin: Admin{
			Username: adminUsername,
			Email:    adminEmail,
			Password: adminPassword,
		},
		HTTP: HTTP{
			Port:              backendPort,
			BaseURL:           apiBaseURL,
			CORSOrigins:       corsOrigins,
			TrustedProxyCIDRs: trustedProxyCIDRs,
			MetricsAllowedIPs: metricsAllowedIPs,
			ShutdownTimeout:   shutdownTimeout,
		},
		DB: DB{
			URL:            dbURL,
			MigrationsPath: migrationsPath,
			MaxConns:       getEnvInt("POSTGRES_MAX_CONNS", 100),
			MinConns:       getEnvInt("POSTGRES_MIN_CONNS", 10),
		},
		JWT: JWT{
			AccessSecret:  jwtAccessSecret,
			RefreshSecret: jwtRefreshSecret,
			AccessTTL:     time.Duration(jwtAccessTTLMin) * time.Minute,
			RefreshTTL:    time.Duration(jwtRefreshTTLHrs) * time.Hour,
		},
		Redis: Redis{
			Host:         redisHost,
			Port:         redisPort,
			Password:     redisPassword,
			PoolSize:     redisPoolSize,
			MinIdleConns: redisMinIdle,
		},
		RateLimit: RateLimit{
			SubmitFlag:         rateLimitSubmitFlag,
			SubmitFlagDuration: rateLimitSubmitFlagDuration,
		},
		Resend: Resend{
			APIKey:      resendAPIKey,
			FromEmail:   resendFromEmail,
			FromName:    resendFromName,
			Enabled:     resendEnabled,
			VerifyTTL:   resendVerifyTTL,
			ResetTTL:    resendResetTTL,
			FrontendURL: frontendURL,
		},
		Storage: Storage{
			Provider:         storageProvider,
			LocalPath:        storageLocalPath,
			S3Endpoint:       storageS3Endpoint,
			S3PublicEndpoint: storageS3PublicEndpoint,
			S3AccessKey:      s3AccessKey,
			S3SecretKey:      s3SecretKey,
			S3Bucket:         storageS3Bucket,
			S3Region:         storageS3Region,
			S3UseSSL:         storageS3UseSSL,
			PresignedExpiry:  storagePresignedExpiry,
		},
		Competition: Competition{
			Mode:            competitionMode,
			AllowTeamSwitch: allowTeamSwitch,
			MinTeamSize:     minTeamSize,
			MaxTeamSize:     maxTeamSize,
		},
		OAuth: OAuth{
			StateSecret: oauthStateSecret,
			GitHub: OAuthProvider{
				ClientID:     oauthGitHubClientID,
				ClientSecret: oauthGitHubClientSecret,
				RedirectURL:  oauthGitHubRedirectURL,
			},
			Google: OAuthProvider{
				ClientID:     oauthGoogleClientID,
				ClientSecret: oauthGoogleClientSecret,
				RedirectURL:  oauthGoogleRedirectURL,
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
