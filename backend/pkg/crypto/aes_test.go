package crypto

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCryptoService(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		key := "12345678901234567890123456789012"
		svc, err := NewCryptoService(key)
		require.NoError(t, err)
		assert.NotNil(t, svc)
	})

	t.Run("Error_InvalidLength", func(t *testing.T) {
		key := "short"
		svc, err := NewCryptoService(key)
		assert.Error(t, err)
		assert.Nil(t, svc)
		assert.Equal(t, "key must be 32 bytes (256 bits) for AES-256", err.Error())
	})
}

func TestCryptoService_EncryptDecrypt_Success(t *testing.T) {
	key := "12345678901234567890123456789012"
	svc, _ := NewCryptoService(key)

	plaintext := "CTF{this_is_a_secret_flag}"

	encrypted, err := svc.Encrypt(plaintext)
	require.NoError(t, err)
	assert.NotEmpty(t, encrypted)
	assert.NotEqual(t, plaintext, encrypted)

	decrypted, err := svc.Decrypt(encrypted)
	require.NoError(t, err)
	assert.Equal(t, plaintext, decrypted)
}

func TestCryptoService_Decrypt_Error(t *testing.T) {
	key := "12345678901234567890123456789012"
	svc, _ := NewCryptoService(key)

	t.Run("InvalidBase64", func(t *testing.T) {
		_, err := svc.Decrypt("invalid_base64")
		assert.Error(t, err)
	})

	t.Run("ShortCiphertext", func(t *testing.T) {
		// Base64 valid but decodes to too few bytes (< nonce size 12)
		short := base64.StdEncoding.EncodeToString([]byte("123"))
		_, err := svc.Decrypt(short)
		assert.Error(t, err)
		assert.Equal(t, "ciphertext too short", err.Error())
	})

	t.Run("TamperedCiphertext", func(t *testing.T) {
		plaintext := "secret"
		encrypted, _ := svc.Encrypt(plaintext)

		// Decode, modify last byte, encode back
		data, _ := base64.StdEncoding.DecodeString(encrypted)
		data[len(data)-1] ^= 0xFF // flip bits
		tampered := base64.StdEncoding.EncodeToString(data)

		_, err := svc.Decrypt(tampered)
		assert.Error(t, err)
	})
}
