package cache

import (
	"testing"

	"github.com/TakuyaYagam1/AstroCTFb/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyScoreboardBracket_Success(t *testing.T) {
	t.Parallel()
	got := KeyScoreboardBracket("bracket-id")
	assert.Equal(t, "scoreboard:bracket:bracket-id", got)
}

func TestKeyScoreboardBracketFrozen_Success(t *testing.T) {
	t.Parallel()
	got := KeyScoreboardBracketFrozen("bracket-id")
	assert.Equal(t, "scoreboard:frozen:bracket:bracket-id", got)
}

func TestNewRedisClient_Error(t *testing.T) {
	t.Parallel()
	_, err := NewRedisClient(&config.Redis{Host: "localhost", Port: "1", Password: ""})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis connection failed")
}
