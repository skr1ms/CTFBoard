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
