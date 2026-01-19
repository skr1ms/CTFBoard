package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	"github.com/skr1ms/CTFBoard/pkg/vault"
)

type (
	Config struct {
		App       `yaml:"app"`
		HTTP      `yaml:"http"`
		DB        `yaml:"mariadb"`
		JWT       `yaml:"jwt"`
		Redis     `yaml:"redis"`
		RateLimit `yaml:"rate_limit"`
		Resend    `yaml:"resend"`
	}

	App struct {
		Name     string
		Version  string
		ChiMode  string
		LogLevel string
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
)

func New() (*Config, error) {
	envPaths := []string{".env", "../.env", "../../.env", "/app/.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			fmt.Printf("Config: .env file loaded successfully from %s\n", path)
			break
		}
	}

	// Initialize ALL variables from Environment first
	appName := getEnv("APP_NAME", "CTFBoard")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	chiMode := getEnv("CHI_MODE", "release")
	logLevel := getEnv("LOG_LEVEL", "info")
	backendPort := getEnv("BACKEND_PORT", "8080")
	migrationsPath := getEnv("MIGRATIONS_PATH", "migrations")
	mariadbHost := getEnv("MARIADB_HOST", "mariadb")
	mariadbPort := getEnv("MARIADB_PORT", "3306")
	redisHost := getEnv("REDIS_HOST", "redis")
	redisPort := getEnv("REDIS_PORT", "6379")
	corsOrigins := parseCORSOrigins(getEnv("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173"))

	mariadbUser := getEnv("MARIADB_USER", "")
	mariadbPassword := getEnv("MARIADB_PASSWORD", "")
	mariadbDB := getEnv("MARIADB_DB", "")
	jwtAccessSecret := getEnv("JWT_ACCESS_SECRET", "")
	jwtRefreshSecret := getEnv("JWT_REFRESH_SECRET", "")
	redisPassword := getEnv("REDIS_PASSWORD", "")
	resendAPIKey := getEnv("RESEND_API_KEY", "")

	l := logger.New(logLevel, chiMode)

	// Try to fetch secrets from Vault and OVERRIDE if successful
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if vaultAddr != "" && vaultToken != "" {
		l.Info("Config: attempting to fetch secrets from Vault", nil)
		vaultClient, err := vault.New(vaultAddr, vaultToken)
		if err == nil {
			// Database secrets
			dbSecrets, err := vaultClient.GetSecret("ctfboard/database")
			if err == nil {
				l.Info("Config: database secrets loaded from Vault", nil)
				if u, ok := dbSecrets["user"].(string); ok && u != "" {
					mariadbUser = u
				}
				if p, ok := dbSecrets["password"].(string); ok && p != "" {
					mariadbPassword = p
				}
				if db, ok := dbSecrets["dbname"].(string); ok && db != "" {
					mariadbDB = db
				}
			} else {
				l.Warn("Config: failed to load database secrets from Vault, using env", err)
			}

			// Redis secrets
			redisSecrets, err := vaultClient.GetSecret("ctfboard/redis")
			if err == nil {
				l.Info("Config: redis secrets loaded from Vault", nil)
				if p, ok := redisSecrets["password"].(string); ok && p != "" {
					redisPassword = p
				}
			} else {
				l.Warn("Config: failed to load redis secrets from Vault, using env", err)
			}

			// JWT secrets
			jwtSecrets, err := vaultClient.GetSecret("ctfboard/jwt")
			if err == nil {
				l.Info("Config: JWT secrets loaded from Vault", nil)
				if access, ok := jwtSecrets["access_secret"].(string); ok && access != "" {
					jwtAccessSecret = access
				}
				if refresh, ok := jwtSecrets["refresh_secret"].(string); ok && refresh != "" {
					jwtRefreshSecret = refresh
				}
			} else {
				l.Warn("Config: failed to load jwt secrets from Vault, using env", err)
			}

			// Resend secrets
			resendSecrets, err := vaultClient.GetSecret("ctfboard/resend")
			if err == nil {
				l.Info("Config: Resend secrets loaded from Vault", nil)
				if k, ok := resendSecrets["api_key"].(string); ok && k != "" {
					resendAPIKey = k
				}
			} else {
				l.Warn("Config: failed to load resend secrets from Vault, using env (or not configured)", err)
			}
		} else {
			l.Error("Config: failed to initialize vault client", err)
		}
	}

	// Final Validation
	if mariadbUser == "" || mariadbPassword == "" || mariadbDB == "" {
		return nil, fmt.Errorf("required database configuration is missing (env or vault)")
	}
	if jwtAccessSecret == "" || jwtRefreshSecret == "" {
		return nil, fmt.Errorf("required jwt configuration is missing (env or vault)")
	}
	if redisPassword == "" {
		return nil, fmt.Errorf("required redis configuration is missing (env or vault)")
	}

	cfg := &Config{
		App: App{
			Name:     appName,
			Version:  appVersion,
			ChiMode:  chiMode,
			LogLevel: logLevel,
		},
		HTTP: HTTP{
			Port:        backendPort,
			CORSOrigins: corsOrigins,
		},
		DB: DB{
			URL:            fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", mariadbUser, mariadbPassword, mariadbHost, mariadbPort, mariadbDB),
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
			SubmitFlag:         getEnvInt("RATE_LIMIT_SUBMIT_FLAG", 10),
			SubmitFlagDuration: time.Duration(getEnvInt("RATE_LIMIT_SUBMIT_FLAG_DURATION", 1)) * time.Minute,
		},
		Resend: Resend{
			APIKey:      resendAPIKey,
			FromEmail:   getEnv("RESEND_FROM_EMAIL", "noreply@ctfboard.local"),
			FromName:    getEnv("RESEND_FROM_NAME", "CTFBoard"),
			Enabled:     getEnvBool("RESEND_ENABLED", false),
			VerifyTTL:   time.Duration(getEnvInt("RESEND_VERIFY_TTL_HOURS", 24)) * time.Hour,
			ResetTTL:    time.Duration(getEnvInt("RESEND_RESET_TTL_HOURS", 1)) * time.Hour,
			FrontendURL: getEnv("FRONTEND_URL", "http://localhost:3000"),
		},
	}

	return cfg, nil
}
