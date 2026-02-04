package helper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (h *E2EHelper) GetUserIDByEmail(email string) string {
	h.t.Helper()
	ctx := context.Background()
	var ID string
	err := h.pool.QueryRow(ctx, "SELECT ID FROM users WHERE email = $1", email).Scan(&ID)
	require.NoError(h.t, err, "failed to find user by email")
	return ID
}

func (h *E2EHelper) AssertUserVerified(email string, expected bool) {
	h.t.Helper()
	ctx := context.Background()
	var isVerified bool
	err := h.pool.QueryRow(ctx, "SELECT is_verified FROM users WHERE email = $1", email).Scan(&isVerified)
	require.NoError(h.t, err)
	assert.Equal(h.t, expected, isVerified, "user verification status mismatch")
}

func (h *E2EHelper) InjectToken(userID string, tokenType entity.TokenType, knownToken string) {
	h.t.Helper()
	ctx := context.Background()
	hashedToken := h.hashToken(knownToken)

	var count int
	err := h.pool.QueryRow(ctx, "SELECT count(*) FROM verification_tokens WHERE user_id = $1 AND type = $2", userID, tokenType).Scan(&count)
	require.NoError(h.t, err)
	require.Equal(h.t, 1, count, "token row should exist before injection")

	_, err = h.pool.Exec(ctx, "UPDATE verification_tokens SET token = $1 WHERE user_id = $2 AND type = $3", hashedToken, userID, tokenType)
	require.NoError(h.t, err, "failed to inject token")
}

func (h *E2EHelper) hashToken(token string) string {
	h.t.Helper()
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
