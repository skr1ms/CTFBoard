package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

const (
	keyVersionByte  = 0
	aes256HexKeyLen = 64
)

var (
	// ErrServiceNotConfigured is returned when encryption operations are attempted on a nil or unconfigured service.
	ErrServiceNotConfigured = errors.New("encryption service not configured")
	// ErrUnknownKeyVersion is returned when the version byte in the ciphertext does not match any known key version.
	ErrUnknownKeyVersion = errors.New("unknown key version")
)

// Service defines the contract for symmetric encryption and decryption of strings.
type Service interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext64 string) (string, error)
}

// CryptoService implements Service using AES-256-GCM with a versioned ciphertext envelope.
type CryptoService struct {
	key []byte
}

// NewCryptoService creates a CryptoService from a 64-character hex-encoded AES-256 key.
func NewCryptoService(
	key string,
) (*CryptoService, error) {
	if len(key) != aes256HexKeyLen {
		return nil, errors.New("key must be 64 characters (hex encoded 32 bytes) for AES-256")
	}

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("crypto.NewCryptoService: invalid hex key: %w", err)
	}

	return &CryptoService{key: keyBytes}, nil
}

// Encrypt encrypts plaintext with AES-256-GCM and returns a base64-encoded string prefixed with a key version byte.
func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("crypto.Encrypt: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.Encrypt: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("crypto.Encrypt: %w", err)
	}

	payload := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	withVersion := make([]byte, 0, 1+len(payload))
	withVersion = append(withVersion, keyVersionByte)
	withVersion = append(withVersion, payload...)

	return base64.StdEncoding.EncodeToString(withVersion), nil
}

// Decrypt decodes a base64-encoded ciphertext produced by Encrypt and returns the original plaintext.
func (s *CryptoService) Decrypt(ciphertext64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext64)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: %w", err)
	}

	if len(data) < 1 {
		return "", errors.New("crypto.Decrypt: ciphertext too short (no version)")
	}

	if data[0] != keyVersionByte {
		return "", ErrUnknownKeyVersion
	}

	data = data[1:]

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("crypto.Decrypt: ciphertext too short (no nonce)")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("crypto.Decrypt: %w", err)
	}

	return string(plaintext), nil
}
