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

const keyVersionByte = 0

var (
	ErrServiceNotConfigured = errors.New("encryption service not configured")
	ErrUnknownKeyVersion    = errors.New("unknown key version")
)

type Service interface {
	Encrypt(plaintext string) (string, error)
	Decrypt(ciphertext64 string) (string, error)
}

type CryptoService struct {
	key []byte
}

func NewCryptoService(
	key string,
) (*CryptoService, error) {
	if len(key) != 64 {
		return nil, errors.New("key must be 64 characters (hex encoded 32 bytes) for AES-256")
	}

	keyBytes, err := hex.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("invalid hex key: %w", err)
	}

	return &CryptoService{key: keyBytes}, nil
}

func (s *CryptoService) Encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	payload := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	withVersion := make([]byte, 0, 1+len(payload))
	withVersion = append(withVersion, keyVersionByte)
	withVersion = append(withVersion, payload...)
	return base64.StdEncoding.EncodeToString(withVersion), nil
}

func (s *CryptoService) Decrypt(ciphertext64 string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext64)
	if err != nil {
		return "", err
	}

	if len(data) < 1 {
		return "", errors.New("ciphertext too short")
	}

	if data[0] != keyVersionByte {
		return "", ErrUnknownKeyVersion
	}
	data = data[1:]

	block, err := aes.NewCipher(s.key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}
