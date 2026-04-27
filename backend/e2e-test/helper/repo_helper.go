package helper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
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

func (h *E2EHelper) InjectToken(userID string, tokenType domain.TokenType, knownToken string) {
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

// BanUserDirectly sets is_banned=true in the DB without going through the usecase,
// so existing JWTs remain valid. Use this in tests where you need a banned user
// but want to keep using the original token (avoids the 1-second iat/revokedAt race).
func (h *E2EHelper) BanUserDirectly(userID string) {
	h.t.Helper()

	_, err := h.pool.Exec(context.Background(),
		"UPDATE users SET is_banned = true, banned_reason = 'test ban', banned_at = now() WHERE id = $1", userID)
	require.NoError(h.t, err, "BanUserDirectly: SQL update failed")

	h.InvalidateUserCache(userID)
}
