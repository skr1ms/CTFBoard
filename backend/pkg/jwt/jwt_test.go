package jwt_test

import (
	"context"
	"testing"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt/mocks"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	testAccessSecret  = "access-secret-at-least-32-bytes!"
	testRefreshSecret = "refresh-secret-at-least-32bytes!"
)

func newTestService(t *testing.T, revoker jwt.RevocationStore) *jwt.JWTService {
	t.Helper()
	svc, err := jwt.NewJWTService(testAccessSecret, testRefreshSecret, time.Hour, time.Hour, revoker, nil)
	require.NoError(t, err)
	return svc
}

func TestJWTService_GenerateTokenPair_Success(t *testing.T) {
	t.Parallel()
	service := newTestService(t, nil)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Greater(t, pair.AccessExpiresAt, time.Now().Unix())
}

func TestJWTService_ValidateAccessToken_Success(t *testing.T) {
	t.Parallel()
	service := newTestService(t, nil)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service.ValidateAccessToken(context.Background(), pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, jwt.TokenTypeAccess, claims.TokenType)
}

func TestJWTService_ValidateAccessToken_InvalidSignature(t *testing.T) {
	t.Parallel()
	service1, err := jwt.NewJWTService("secret-1-at-least-32-bytes-long!", "refresh-1-at-least-32-bytes-lon!", time.Hour, time.Hour, nil, nil)
	require.NoError(t, err)
	service2, err := jwt.NewJWTService("secret-2-at-least-32-bytes-long!", "refresh-2-at-least-32-bytes-lon!", time.Hour, time.Hour, nil, nil)
	require.NoError(t, err)
	userID := uuid.New()

	pair, err := service1.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service2.ValidateAccessToken(context.Background(), pair.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateRefreshToken_Success(t *testing.T) {
	t.Parallel()
	service := newTestService(t, nil)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, jwt.TokenTypeRefresh, claims.TokenType)
}

func TestJWTService_RefreshTokens_Success(t *testing.T) {
	t.Parallel()
	service := newTestService(t, nil)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	newPair, err := service.RefreshTokens(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
}

func TestJWTService_RefreshTokens_InvalidToken(t *testing.T) {
	t.Parallel()
	service := newTestService(t, nil)

	newPair, err := service.RefreshTokens(context.Background(), "invalid-token")
	assert.Error(t, err)
	assert.Nil(t, newPair)
	assert.Contains(t, err.Error(), "validate refresh token")
}

func TestJWTService_NewJWTService_ShortSecret(t *testing.T) {
	t.Parallel()
	_, err := jwt.NewJWTService("short", "short", time.Hour, time.Hour, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "at least")
}

func TestJWTService_RefreshTokens_RevokesOldToken(t *testing.T) {
	t.Parallel()
	revoker := mocks.NewMockRevocationStore(t)

	revoked := make(map[string]bool)
	revoker.EXPECT().
		Revoke(mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
		RunAndReturn(func(_ context.Context, jti string, _ time.Duration) error {
			revoked[jti] = true
			return nil
		})
	revoker.EXPECT().
		IsRevoked(mock.Anything, mock.AnythingOfType("string")).
		RunAndReturn(func(_ context.Context, jti string) (bool, error) {
			return revoked[jti], nil
		}).
		Maybe()

	service := newTestService(t, revoker)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	_, err = service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)

	claims, err := service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	require.NoError(t, err)
	oldJTI := claims.ID

	newPair, err := service.RefreshTokens(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newPair.RefreshToken)

	_, err = service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")

	assert.True(t, revoked[oldJTI])
}

func TestJWTService_RevokeRefreshToken_ThenValidateFails(t *testing.T) {
	t.Parallel()
	revoker := mocks.NewMockRevocationStore(t)

	revoked := make(map[string]bool)
	revoker.EXPECT().
		Revoke(mock.Anything, mock.AnythingOfType("string"), mock.AnythingOfType("time.Duration")).
		RunAndReturn(func(_ context.Context, jti string, _ time.Duration) error {
			revoked[jti] = true
			return nil
		})
	revoker.EXPECT().
		IsRevoked(mock.Anything, mock.AnythingOfType("string")).
		RunAndReturn(func(_ context.Context, jti string) (bool, error) {
			return revoked[jti], nil
		}).
		Maybe()

	service := newTestService(t, revoker)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	_, err = service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)

	err = service.RevokeRefreshToken(context.Background(), pair.RefreshToken)
	assert.NoError(t, err)

	_, err = service.ValidateRefreshToken(context.Background(), pair.RefreshToken)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}
