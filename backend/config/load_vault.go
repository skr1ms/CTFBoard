package config

import (
	"context"
	"os"
	"sync"

	"github.com/wahrwelt-kit/go-logkit"
	"golang.org/x/sync/errgroup"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/vault"
)

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
