package config

import (
	"github.com/ilyakaznacheev/cleanenv"
	"github.com/joho/godotenv"
	"github.com/wahrwelt-kit/go-logkit"
)

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
	raw.TrustedProxyCIDRs = parseTrustedProxyCIDRs(raw.TrustedProxyCIDRsStr)

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
