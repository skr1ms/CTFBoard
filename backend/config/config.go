package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/vault"
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
		Port        string
		CORSOrigins []string
	}

	DB struct {
		URL            string
		MigrationsPath string
	}

	JWT struct {
		AccessSecret  string
		RefreshSecret string
		AccessTTL     time.Duration
		RefreshTTL    time.Duration
	}

	Redis struct {
		Host     string
		Port     string
		Password string
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
		S3UseSSL         bool
		PresignedExpiry  time.Duration
	}

	Competition struct {
		Mode            string
		AllowTeamSwitch bool
		MinTeamSize     int
		MaxTeamSize     int
	}
)

//nolint:gocognit,gocyclo,funlen
func New() (*Config, error) {
	envPaths := []string{".env", "../.env", "../../.env", "/app/.env"}
	envLoaded := false
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			fmt.Printf("Config: .env file loaded from %s\n", path)
			envLoaded = true
			break
		}
	}

	if !envLoaded {
		fmt.Println("Config: .env file not found, using environment variables (production mode)")
	}

	// Initialize ALL variables from Environment first
	appName := getEnv("APP_NAME", "CTFBoard")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	chiMode := getEnv("CHI_MODE", "release")
	logLevel := getEnv("LOG_LEVEL", "info")
	flagEncryptionKey := getEnv("FLAG_ENCRYPTION_KEY", "")
	verifyEmails := getEnvBool("VERIFY_EMAILS", false)
	backendPort := getEnv("BACKEND_PORT", "8080")
	migrationsPath := getEnv("MIGRATIONS_PATH", "migrations")
	corsOrigins := parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173,http://localhost:5000"))

	postgresHost := getEnv("POSTGRES_HOST", "postgres")
	postgresPort := getEnv("POSTGRES_PORT", "5432")
	postgresUser := getEnv("POSTGRES_USER", "")
	postgresPassword := getEnv("POSTGRES_PASSWORD", "")
	postgresDB := getEnv("POSTGRES_DB", "")

	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	redisPassword := getEnv("REDIS_PASSWORD", "")

	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")

	resendAPIKey := getEnv("RESEND_API_KEY", "")
	s3AccessKey := getEnv("STORAGE_S3_ACCESS_KEY", "")
	s3SecretKey := getEnv("STORAGE_S3_SECRET_KEY", "")

	adminUsername := getEnv("ADMIN_USERNAME", "")
	adminEmail := getEnv("ADMIN_EMAIL", "")
	adminPassword := getEnv("ADMIN_PASSWORD", "")

	rateLimitSubmitFlag := getEnvInt("RATE_LIMIT_SUBMIT_FLAG", 10)
	rateLimitSubmitFlagDuration := time.Duration(getEnvInt("RATE_LIMIT_SUBMIT_FLAG_DURATION", 1)) * time.Minute

	resendFromEmail := getEnv("RESEND_FROM_EMAIL", "noreply@ctfboard.local")
	resendFromName := getEnv("RESEND_FROM_NAME", "CTFBoard")
	resendEnabled := getEnvBool("RESEND_ENABLED", false)
	resendVerifyTTL := time.Duration(getEnvInt("RESEND_VERIFY_TTL_HOURS", 24)) * time.Hour
	resendResetTTL := time.Duration(getEnvInt("RESEND_RESET_TTL_HOURS", 1)) * time.Hour
	frontendURL := getEnv("FRONTEND_URL", "http://localhost:3000")

	storageProvider := getEnv("STORAGE_PROVIDER", "filesystem")
	storageLocalPath := getEnv("STORAGE_LOCAL_PATH", "./uploads")
	storageS3Endpoint := getEnv("STORAGE_S3_ENDPOINT", "urchin:9000")
	storageS3PublicEndpoint := getEnv("STORAGE_S3_PUBLIC_ENDPOINT", "")
	storageS3Bucket := getEnv("STORAGE_S3_BUCKET", "tasks")
	storageS3UseSSL := getEnvBool("STORAGE_S3_USE_SSL", false)
	storagePresignedExpiry := time.Duration(getEnvInt("STORAGE_PRESIGNED_EXPIRY_MINUTES", 60)) * time.Minute

	competitionMode := getEnv("COMPETITION_MODE", "flexible")
	allowTeamSwitch := getEnvBool("ALLOW_TEAM_SWITCH", true)
	minTeamSize := getEnvInt("MIN_TEAM_SIZE", 1)
	maxTeamSize := getEnvInt("MAX_TEAM_SIZE", 10)

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
			// Database secrets
			dbSecrets, err := vaultClient.GetSecret("ctfboard/database")
			if err == nil {
				l.Info("Config: database secrets loaded from Vault")
				if u, ok := dbSecrets[entity.RoleUser].(string); ok && u != "" {
					postgresUser = u
				}
				if p, ok := dbSecrets["password"].(string); ok && p != "" {
					postgresPassword = p
				}
				if db, ok := dbSecrets["dbname"].(string); ok && db != "" {
					postgresDB = db
				}
			} else {
				l.WithError(err).Warn("Config: failed to load database secrets from Vault, using env")
			}

			// Redis secrets
			redisSecrets, err := vaultClient.GetSecret("ctfboard/redis")
			if err == nil {
				l.Info("Config: redis secrets loaded from Vault")
				if p, ok := redisSecrets["password"].(string); ok && p != "" {
					redisPassword = p
				}
			} else {
				l.WithError(err).Warn("Config: failed to load redis secrets from Vault, using env")
			}

			// JWT secrets
			jwtSecrets, err := vaultClient.GetSecret("ctfboard/jwt")
			if err == nil {
				l.Info("Config: JWT secrets loaded from Vault")
				if access, ok := jwtSecrets["access_secret"].(string); ok && access != "" {
					jwtAccessSecret = access
				}
				if refresh, ok := jwtSecrets["refresh_secret"].(string); ok && refresh != "" {
					jwtRefreshSecret = refresh
				}
			} else {
				l.WithError(err).Warn("Config: failed to load jwt secrets from Vault, using env")
			}

			// Resend secrets
			resendSecrets, err := vaultClient.GetSecret("ctfboard/resend")
			if err == nil {
				l.Info("Config: Resend secrets loaded from Vault")
				if k, ok := resendSecrets["api_key"].(string); ok && k != "" {
					resendAPIKey = k
				}
			} else {
				l.WithError(err).Warn("Config: failed to load resend secrets from Vault, using env (or not configured)")
			}

			// Storage secrets
			storageSecrets, err := vaultClient.GetSecret("ctfboard/storage")
			if err == nil {
				l.Info("Config: Storage secrets loaded from Vault")
				if k, ok := storageSecrets["access_key"].(string); ok && k != "" {
					s3AccessKey = k
				}
				if s, ok := storageSecrets["secret_key"].(string); ok && s != "" {
					s3SecretKey = s
				}
			} else {
				l.WithError(err).Warn("Config: failed to load storage secrets from Vault (optional)")
			}

			// App secrets (encryption keys)
			appSecrets, err := vaultClient.GetSecret("ctfboard/app")
			if err == nil {
				l.Info("Config: app secrets loaded from Vault")
				if key, ok := appSecrets["flag_encryption_key"].(string); ok && key != "" {
					flagEncryptionKey = key
				}
			} else {
				l.WithError(err).Warn("Config: failed to load app secrets from Vault, using env")
			}

			// Admin secrets (default admin credentials)
			adminSecrets, err := vaultClient.GetSecret("ctfboard/admin")
			if err == nil {
				l.Info("Config: admin secrets loaded from Vault")
				if u, ok := adminSecrets["username"].(string); ok && u != "" {
					adminUsername = u
				}
				if e, ok := adminSecrets["email"].(string); ok && e != "" {
					adminEmail = e
				}
				if p, ok := adminSecrets["password"].(string); ok && p != "" {
					adminPassword = p
				}
			} else {
				l.WithError(err).Warn("Config: failed to load admin secrets from Vault, using env (optional)")
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
	if redisPassword == "" {
		return nil, fmt.Errorf("required redis configuration is missing (env or vault)")
	}
	if flagEncryptionKey == "" {
		return nil, fmt.Errorf("required flag encryption key is missing (env or vault) - needed for regex challenges")
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)

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
			Port:        backendPort,
			CORSOrigins: corsOrigins,
		},
		DB: DB{
			URL:            dbURL,
			MigrationsPath: migrationsPath,
		},
		JWT: JWT{
			AccessSecret:  jwtAccessSecret,
			RefreshSecret: jwtRefreshSecret,
			AccessTTL:     1 * 24 * time.Hour,
			RefreshTTL:    3 * 24 * time.Hour,
		},
		Redis: Redis{
			Host:     redisHost,
			Port:     redisPort,
			Password: redisPassword,
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
			S3UseSSL:         storageS3UseSSL,
			PresignedExpiry:  storagePresignedExpiry,
		},
		Competition: Competition{
			Mode:            competitionMode,
			AllowTeamSwitch: allowTeamSwitch,
			MinTeamSize:     minTeamSize,
			MaxTeamSize:     maxTeamSize,
		},
	}

	return cfg, nil
}
