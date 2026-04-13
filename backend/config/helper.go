package config

import (
	"context"
	"net"
	"strings"

	"github.com/samber/lo"
	"github.com/wahrwelt-kit/go-logkit"
)

type vaultSecretGetter interface {
	GetSecret(ctx context.Context, path string) (map[string]any, error)
}

// vaultFetch returns an errgroup-compatible closure that fetches the Vault KV secret
// at path and calls apply (mutex-wrapped by the caller) to write selected fields into
// rawConfig. On error it logs a warning and returns nil so that a missing or
// inaccessible secret path does not abort the entire errgroup - the field simply
// retains its env/default value (errSuffix describes that fallback in the log line).
func vaultFetch(ctx context.Context, client vaultSecretGetter, l logkit.Logger, path, logName, errSuffix string, apply func(map[string]any)) func() error {
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

func parseCORSOrigins(s string) []string {
	if s == "" {
		return []string{}
	}

	return lo.Map(strings.Split(s, ","), func(x string, _ int) string { return strings.TrimSpace(x) })
}

func parseCommaSeparated(s string) []string {
	if s == "" {
		return nil
	}

	return lo.Filter(lo.Map(strings.Split(s, ","), func(x string, _ int) string { return strings.TrimSpace(x) }), func(s string, _ int) bool { return s != "" })
}

// parseTrustedProxyCIDRs parses a comma-separated list of CIDR strings. Invalid
// entries are silently dropped with a warning log rather than failing startup,
// because misconfigured trusted-proxy CIDRs should not block the server - but
// they do affect IP-based access control, so the warning is intentionally noisy.
func parseTrustedProxyCIDRs(s string, l logkit.Logger) []string {
	if s == "" {
		return nil
	}

	trimmed := lo.Map(strings.Split(s, ","), func(x string, _ int) string { return strings.TrimSpace(x) })

	return lo.FilterMap(trimmed, func(p string, _ int) (string, bool) {
		if p == "" {
			return "", false
		}

		if _, _, err := net.ParseCIDR(p); err != nil {
			l.WithError(err).Warn("Config: TRUSTED_PROXY_CIDRS invalid CIDR, skipping", logkit.Fields{"cidr": p})

			return "", false
		}

		return p, true
	})
}
