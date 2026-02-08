package postgres

import (
	"testing"

	"github.com/skr1ms/CTFBoard/config"
	"github.com/stretchr/testify/require"
)

func TestNew_InvalidURL_Error(t *testing.T) {
	cfg := &config.DB{URL: "://invalid"}
	pool, err := New(cfg)
	require.Error(t, err)
	require.Nil(t, pool)
}

func TestNew_InvalidDSN_Error(t *testing.T) {
	cfg := &config.DB{URL: "postgres://%zz@localhost/db"}
	pool, err := New(cfg)
	require.Error(t, err)
	require.Nil(t, pool)
}
