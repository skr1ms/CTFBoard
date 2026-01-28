package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo/persistent"
	"github.com/skr1ms/CTFBoard/internal/usecase"
	"github.com/skr1ms/CTFBoard/pkg/vault"
)

func main() {
	envPaths := []string{".env", "../.env", "/app/.env", "/etc/ctfboard/.env"}
	for _, path := range envPaths {
		if err := godotenv.Load(path); err == nil {
			log.Printf("Loaded .env from %s", path)
			break
		}
	}

	host := getEnv("POSTGRES_HOST", "localhost")
	port := getEnv("POSTGRES_PORT", "5432")
	user := getEnv("POSTGRES_USER", "postgres")
	password := getEnv("POSTGRES_PASSWORD", "postgres")
	dbname := getEnv("POSTGRES_DB", "ctfboard")

	vaultAddr := os.Getenv("VAULT_ADDR")
	vaultToken := os.Getenv("VAULT_TOKEN")

	if vaultAddr != "" && vaultToken != "" {
		log.Println("Attempting to fetch secrets from Vault...")
		vaultClient, err := vault.New(vaultAddr, vaultToken)
		if err == nil {
			dbSecrets, err := vaultClient.GetSecret("ctfboard/database")
			if err == nil {
				log.Println("Database secrets loaded from Vault")
				if u, ok := dbSecrets[entity.RoleUser].(string); ok && u != "" {
					user = u
				}
				if p, ok := dbSecrets["password"].(string); ok && p != "" {
					password = p
				}
				if db, ok := dbSecrets["dbname"].(string); ok && db != "" {
					dbname = db
				}
			} else {
				log.Printf("Failed to load database secrets from Vault: %v", err)
			}
		} else {
			log.Printf("Failed to initialize vault client: %v", err)
		}
	}

	dbURL := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port, dbname)

	ctx := context.Background()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v\n", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	teamRepo := persistent.NewTeamRepo(pool)
	cleanupUC := usecase.NewCleanupUseCase(teamRepo)

	duration := 30 * 24 * time.Hour
	log.Printf("Starting cleanup of teams deleted more than %v ago", duration)

	if err := cleanupUC.CleanupDeletedTeams(ctx, duration); err != nil {
		log.Fatalf("Cleanup failed: %v", err)
	}

	log.Println("Cleanup completed successfully")
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
