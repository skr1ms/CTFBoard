package migrator

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/config"
)

func TestRun_InvalidMigrationsPath_Error(t *testing.T) {
	cfg := &config.DB{URL: "postgres://user:pass@localhost:5432/db?sslmode=disable", MigrationsPath: "/nonexistent-migrations"}
	err := Run(cfg)
	require.Error(t, err)
}

func TestRun_InvalidDBURL_Error(t *testing.T) {
	cfg := &config.DB{URL: "postgres://invalid-%zz@localhost/db", MigrationsPath: "."}
	err := Run(cfg)
	require.Error(t, err)
}
