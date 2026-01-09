package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/skr1ms/CTFBoard/pkg/vault"
)

type (
	Config struct {
		App  `yaml:"app"`
		HTTP `yaml:"http"`
		DB   `yaml:"mariadb"`
		JWT  `yaml:"jwt"`
	}

	App struct {
		Name     string
		Version  string
		ChiMode  string
		LogLevel string
	}

	HTTP struct {
		Port string
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
)

func New() (*Config, error) {
	_ = godotenv.Load()

	var vaultClient *vault.Client
	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if vaultAddr != "" && vaultToken != "" {
		var err error
		vaultClient, err = vault.New(vaultAddr, vaultToken)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize vault client: %w", err)
		}
	}

	// Non-sensitive configuration from environment variables
	appName := getEnv("APP_NAME", "CTFBoard")
	appVersion := getEnv("APP_VERSION", "1.0.0")
	chiMode := getEnv("CHI_MODE", "release")
	logLevel := getEnv("LOG_LEVEL", "info")
	backendPort := getEnv("BACKEND_PORT", "8080")
	migrationsPath := getEnv("MIGRATIONS_PATH", "migrations")
	mariadbHost := getEnv("MARIADB_HOST", "mariadb")
	mariadbPort := getEnv("MARIADB_PORT", "3306")

	// Sensitive secrets: try Vault first, fallback to environment variables
	var mariadbUser, mariadbPassword, mariadbDB string
	var jwtAccessSecret, jwtRefreshSecret string

	if vaultClient != nil {
		log.Println("Config: attempting to fetch secrets from Vault")

		// Database secrets from Vault
		dbSecrets, err := vaultClient.GetSecret("ctfboard/database")
		if err != nil {
			return nil, fmt.Errorf("failed to load database secrets from vault: %w", err)
		}

		log.Println("Config: database secrets loaded from Vault")

		if u, ok := dbSecrets["user"].(string); ok && u != "" {
			mariadbUser = u
		} else {
			return nil, fmt.Errorf("database user not found in vault secret")
		}
		if p, ok := dbSecrets["password"].(string); ok && p != "" {
			mariadbPassword = p
		} else {
			return nil, fmt.Errorf("database password not found in vault secret")
		}
		if db, ok := dbSecrets["dbname"].(string); ok && db != "" {
			mariadbDB = db
		} else {
			return nil, fmt.Errorf("database name not found in vault secret")
		}

		// JWT secrets from Vault
		jwtSecrets, err := vaultClient.GetSecret("ctfboard/jwt")
		if err != nil {
			return nil, fmt.Errorf("failed to load jwt secrets from vault: %w", err)
		}

		log.Println("Config: JWT secrets loaded from Vault")

		if access, ok := jwtSecrets["access_secret"].(string); ok && access != "" {
			jwtAccessSecret = access
		} else {
			return nil, fmt.Errorf("jwt access secret not found in vault secret")
		}
		if refresh, ok := jwtSecrets["refresh_secret"].(string); ok && refresh != "" {
			jwtRefreshSecret = refresh
		} else {
			return nil, fmt.Errorf("jwt refresh secret not found in vault secret")
		}
	} else {
		// Fallback to environment variables if Vault is not available
		mariadbUser = getEnv("MARIADB_USER", "")
		mariadbPassword = getEnv("MARIADB_PASSWORD", "")
		mariadbDB = getEnv("MARIADB_DB", "")
		jwtAccessSecret = getEnv("JWT_ACCESS_SECRET", "")
		jwtRefreshSecret = getEnv("JWT_REFRESH_SECRET", "")

		if mariadbUser == "" || mariadbPassword == "" || mariadbDB == "" {
			return nil, fmt.Errorf("vault not available and required database environment variables are missing")
		}
		if jwtAccessSecret == "" || jwtRefreshSecret == "" {
			return nil, fmt.Errorf("vault not available and required jwt environment variables are missing")
		}
	}

	cfg := &Config{
		App: App{
			Name:     appName,
			Version:  appVersion,
			ChiMode:  chiMode,
			LogLevel: logLevel,
		},
		HTTP: HTTP{
			Port: backendPort,
		},
		DB: DB{
			// For PostgreSQL: change URL format to postgres://user:password@host:port/dbname?sslmode=disable
			// Example: fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", postgresUser, postgresPassword, postgresHost, postgresPort, postgresDB)
			URL:            fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", mariadbUser, mariadbPassword, mariadbHost, mariadbPort, mariadbDB),
			MigrationsPath: migrationsPath,
		},
		JWT: JWT{
			AccessSecret:  jwtAccessSecret,
			RefreshSecret: jwtRefreshSecret,
			AccessTTL:     1 * 24 * time.Hour,
			RefreshTTL:    3 * 24 * time.Hour,
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	return defaultValue
}
