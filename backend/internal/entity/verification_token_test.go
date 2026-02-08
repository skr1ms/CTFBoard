package entity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestVerificationToken_IsExpired_Success(t *testing.T) {
	tok := &VerificationToken{ExpiresAt: time.Now().Add(-time.Hour)}
	assert.True(t, tok.IsExpired())
}

func TestVerificationToken_IsExpired_Error(t *testing.T) {
	tok := &VerificationToken{ExpiresAt: time.Now().Add(time.Hour)}
	assert.False(t, tok.IsExpired())
}

func TestVerificationToken_IsUsed_Success(t *testing.T) {
	used := time.Now()
	tok := &VerificationToken{UsedAt: &used}
	assert.True(t, tok.IsUsed())
}

func TestVerificationToken_IsUsed_Error(t *testing.T) {
	tok := &VerificationToken{UsedAt: nil}
	assert.False(t, tok.IsUsed())
}
