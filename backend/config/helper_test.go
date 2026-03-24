package config

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wahrwelt-kit/go-logkit"

	configMock "github.com/TakuyaYagam1/AstroCTFb/config/mock"
)

func TestParseCORSOrigins_Success(t *testing.T) {
	got := parseCORSOrigins("http://a.com, http://b.com ,https://c.com")
	assert.Equal(t, []string{"http://a.com", "http://b.com", "https://c.com"}, got)
}

func TestParseCORSOrigins_Empty(t *testing.T) {
	got := parseCORSOrigins("")
	assert.Empty(t, got)
}

func TestVaultFetch_Success(t *testing.T) {
	applied := false
	client := configMock.NewMockVaultSecretGetter(t)
	client.EXPECT().GetSecret(mock.Anything, "test/path").Return(map[string]any{"key": "value"}, nil)
	l, logErr := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, logErr)
	fn := vaultFetch(context.Background(), client, l, "test/path", "test", "fallback", func(s map[string]any) {
		applied = true
		assert.Equal(t, "value", s["key"])
	})
	err := fn()
	require.NoError(t, err)
	assert.True(t, applied)
}

func TestVaultFetch_Error(t *testing.T) {
	applied := false
	client := configMock.NewMockVaultSecretGetter(t)
	client.EXPECT().GetSecret(mock.Anything, mock.Anything).Return(nil, errors.New("vault error"))
	l, logErr := logkit.New(logkit.WithLevel(logkit.InfoLevel), logkit.WithOutput(logkit.ConsoleOutput))
	require.NoError(t, logErr)
	fn := vaultFetch(context.Background(), client, l, "test/path", "test", "fallback", func(map[string]any) {
		applied = true
	})
	err := fn()
	require.NoError(t, err)
	assert.False(t, applied)
}
