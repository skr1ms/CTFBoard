package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateCORSOrigins(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		origins []string
		wantErr string
	}{
		{
			name:    "valid exact origins",
			origins: []string{"http://localhost:3000", "https://example.com", "https://example.com:8443"},
		},
		{
			name:    "wildcard",
			origins: []string{"*"},
			wantErr: "wildcard origin",
		},
		{
			name:    "wildcard subdomain",
			origins: []string{"https://*.example.com"},
			wantErr: "wildcard origin",
		},
		{
			name:    "missing scheme",
			origins: []string{"example.com"},
			wantErr: "must use http or https scheme",
		},
		{
			name:    "invalid scheme",
			origins: []string{"ftp://example.com"},
			wantErr: "must use http or https scheme",
		},
		{
			name:    "path",
			origins: []string{"https://example.com/app"},
			wantErr: "without user info, path, query, or fragment",
		},
		{
			name:    "query",
			origins: []string{"https://example.com?x=1"},
			wantErr: "without user info, path, query, or fragment",
		},
		{
			name:    "user info",
			origins: []string{"https://user@example.com"},
			wantErr: "without user info, path, query, or fragment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateCORSOrigins(tt.origins)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidateRejectsWildcardCORSOrigins(t *testing.T) {
	t.Parallel()

	raw := validRawConfigForValidation()
	raw.CORSOrigins = []string{"*"}

	err := validate(raw)

	require.Error(t, err)
	require.Contains(t, err.Error(), "wildcard origin")
}

func TestValidateTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cidrs   []string
		wantErr string
	}{
		{
			name:  "valid private cidr",
			cidrs: []string{"172.16.0.0/12"},
		},
		{
			name:    "invalid cidr",
			cidrs:   []string{"not-a-cidr"},
			wantErr: "invalid CIDR",
		},
		{
			name:    "all ipv4",
			cidrs:   []string{"0.0.0.0/0"},
			wantErr: "must not trust all addresses",
		},
		{
			name:    "all ipv6",
			cidrs:   []string{"::/0"},
			wantErr: "must not trust all addresses",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateTrustedProxyCIDRs(tt.cidrs)
			if tt.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func TestValidateRejectsInvalidTrustedProxyCIDRs(t *testing.T) {
	t.Parallel()

	raw := validRawConfigForValidation()
	raw.TrustedProxyCIDRs = []string{"not-a-cidr"}

	err := validate(raw)

	require.Error(t, err)
	require.Contains(t, err.Error(), "TRUSTED_PROXY_CIDRS contains invalid CIDR")
}

func TestValidateRejectsInvalidConnectionAndStorageBounds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(*rawConfig)
		wantErr string
	}{
		{
			name: "postgres min greater than max",
			mutate: func(raw *rawConfig) {
				raw.PostgresMinConns = 11
				raw.PostgresMaxConns = 10
			},
			wantErr: "POSTGRES_MIN_CONNS=11 must be >= 0 and <= POSTGRES_MAX_CONNS=10",
		},
		{
			name: "redis pool size",
			mutate: func(raw *rawConfig) {
				raw.RedisPoolSize = 0
			},
			wantErr: "REDIS_POOL_SIZE must be a positive integer",
		},
		{
			name: "redis min idle greater than pool",
			mutate: func(raw *rawConfig) {
				raw.RedisMinIdle = 6
				raw.RedisPoolSize = 5
			},
			wantErr: "REDIS_MIN_IDLE=6 must be >= 0 and <= REDIS_POOL_SIZE=5",
		},
		{
			name: "redis port out of range",
			mutate: func(raw *rawConfig) {
				raw.RedisPort = "70000"
			},
			wantErr: "REDIS_PORT must be between 1 and 65535",
		},
		{
			name: "filesystem path empty",
			mutate: func(raw *rawConfig) {
				raw.StorageLocalPath = ""
			},
			wantErr: "STORAGE_LOCAL_PATH must be set",
		},
		{
			name: "s3 public endpoint invalid scheme",
			mutate: func(raw *rawConfig) {
				raw.S3PublicEndpoint = "ftp://cdn.example.com"
			},
			wantErr: "STORAGE_S3_PUBLIC_ENDPOINT must use http or https scheme",
		},
		{
			name: "s3 public endpoint with path",
			mutate: func(raw *rawConfig) {
				raw.S3PublicEndpoint = "https://cdn.example.com/minio"
			},
			wantErr: "STORAGE_S3_PUBLIC_ENDPOINT must be an exact origin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			raw := validRawConfigForValidation()
			tt.mutate(raw)

			err := validate(raw)

			require.Error(t, err)
			require.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

func validRawConfigForValidation() *rawConfig {
	return &rawConfig{
		PostgresUser:                "postgres",
		PostgresPassword:            "postgres",
		PostgresDB:                  "astroctfb",
		JWTAccessSecret:             strings.Repeat("a", 32),
		JWTRefreshSecret:            strings.Repeat("b", 32),
		RedisPassword:               "redis",
		FlagEncryptionKey:           strings.Repeat("0", flagEncryptionKeyHexLen),
		SetupToken:                  strings.Repeat("s", 32),
		CompetitionMode:             "teams_only",
		MinTeamSize:                 1,
		MaxTeamSize:                 5,
		StorageProvider:             "filesystem",
		RateLimitSubmitFlag:         5,
		RateLimitSubmitFlagDuration: 60,
		JWTAccessTTLMin:             15,
		JWTRefreshTTLHrs:            24,
		ResendVerifyTTLHrs:          24,
		ResendResetTTLHrs:           2,
		StoragePresignedExpiryMin:   10,
		CORSOrigins:                 []string{"http://localhost:3000"},
		BackendPort:                 "8080",
		PostgresPort:                "5432",
		PostgresMaxConns:            100,
		PostgresMinConns:            10,
		RedisPort:                   "6379",
		RedisPoolSize:               50,
		RedisMinIdle:                10,
		StorageLocalPath:            "./uploads",
	}
}
