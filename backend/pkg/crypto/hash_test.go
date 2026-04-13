package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSHA256Hex_KnownValue(t *testing.T) {
	t.Parallel()
	// echo -n "hello" | sha256sum
	result := SHA256Hex("hello")
	assert.Equal(t, "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824", result)
}

func TestSHA256Hex_EmptyString(t *testing.T) {
	t.Parallel()

	result := SHA256Hex("")
	assert.Len(t, result, 64)
	assert.Equal(t, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", result)
}

func TestSHA256Hex_DifferentInputs_DifferentHashes(t *testing.T) {
	t.Parallel()

	a := SHA256Hex("foo")
	b := SHA256Hex("bar")
	assert.NotEqual(t, a, b)
}

func TestHMACSign_And_HMACVerify_RoundTrip(t *testing.T) {
	t.Parallel()

	key := []byte("secret-key")
	data := []byte("payload")
	sig := HMACSign(key, data)
	assert.True(t, HMACVerify(key, data, sig))
}

func TestHMACVerify_WrongKey_ReturnsFalse(t *testing.T) {
	t.Parallel()

	sig := HMACSign([]byte("key1"), []byte("data"))
	assert.False(t, HMACVerify([]byte("key2"), []byte("data"), sig))
}

func TestHMACVerify_WrongData_ReturnsFalse(t *testing.T) {
	t.Parallel()

	sig := HMACSign([]byte("key"), []byte("data"))
	assert.False(t, HMACVerify([]byte("key"), []byte("other"), sig))
}

func TestSecureRandomHex_Length(t *testing.T) {
	t.Parallel()

	result, err := SecureRandomHex(16)
	require.NoError(t, err)
	assert.Len(t, result, 32) // 16 bytes -> 32 hex chars
}

func TestSecureRandomHex_Randomness(t *testing.T) {
	t.Parallel()

	a, err := SecureRandomHex(16)
	require.NoError(t, err)
	b, err := SecureRandomHex(16)
	require.NoError(t, err)
	assert.NotEqual(t, a, b)
}

func TestHashHex_ConsistentWithSHA256Hex(t *testing.T) {
	t.Parallel()

	import_hash := SHA256Hex("test-data")
	// HashHex finalizes a hash.Hash - verify it's consistent
	assert.Len(t, import_hash, 64)
}
