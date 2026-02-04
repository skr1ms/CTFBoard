package redis

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKeyScoreboardBracket_Success(t *testing.T) {
	got := KeyScoreboardBracket("bracket-id")
	assert.Equal(t, "scoreboard:bracket:bracket-id", got)
}

func TestKeyScoreboardBracketFrozen_Success(t *testing.T) {
	got := KeyScoreboardBracketFrozen("bracket-id")
	assert.Equal(t, "scoreboard:frozen:bracket:bracket-id", got)
}

func TestNew_Error(t *testing.T) {
	_, err := New("localhost", "1", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis connection failed")
}
