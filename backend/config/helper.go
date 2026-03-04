package config

import (
	"context"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

type vaultSecretGetter interface {
	GetSecret(ctx context.Context, path string) (map[string]any, error)
}

func vaultFetch(ctx context.Context, client vaultSecretGetter, l logger.Logger, path, logName, errSuffix string, apply func(map[string]any)) func() error {
	return func() error {
		s, err := client.GetSecret(ctx, path)
		if err != nil {
			l.WithError(err).Warn("Config: failed to load " + logName + " secrets from Vault, " + errSuffix)
			return nil
		}
		l.Info("Config: " + logName + " secrets loaded from Vault")
		apply(s)
		return nil
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return strings.TrimSpace(value)
	}
	log.Printf("[config] %s not set, using default: %q", key, defaultValue)
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			log.Printf("[config] %s has invalid integer value, using default: %d", key, defaultValue)
			return defaultValue
		}
		return intValue
	}
	log.Printf("[config] %s not set, using default: %d", key, defaultValue)
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		boolValue, err := strconv.ParseBool(value)
		if err != nil {
			log.Printf("[config] %s has invalid boolean value, using default: %v", key, defaultValue)
			return defaultValue
		}
		return boolValue
	}
	log.Printf("[config] %s not set, using default: %v", key, defaultValue)
	return defaultValue
}

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{}
	}
	origins := strings.Split(s, ",")
	for i := range origins {
		origins[i] = strings.TrimSpace(origins[i])
	}
	return origins
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func parseTrustedProxyCIDRs(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		_, _, err := net.ParseCIDR(p)
		if err != nil {
			log.Printf("[config] TRUSTED_PROXY_CIDRS: invalid CIDR %q, skipping: %v", p, err)
			continue
		}
		out = append(out, p)
	}
	return out
}
