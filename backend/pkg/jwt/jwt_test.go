package jwt_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTService_GenerateTokenPair_Success(t *testing.T) {
	service := jwt.NewJWTService("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	assert.NoError(t, err)
	assert.NotEmpty(t, pair.AccessToken)
	assert.NotEmpty(t, pair.RefreshToken)
	assert.Greater(t, pair.AccessExpiresAt, time.Now().Unix())
}

func TestJWTService_ValidateAccessToken_Success(t *testing.T) {
	service := jwt.NewJWTService("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service.ValidateAccessToken(pair.AccessToken)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, "test@example.com", claims.Email)
	assert.Equal(t, jwt.TokenTypeAccess, claims.TokenType)
}

func TestJWTService_ValidateAccessToken_InvalidSignature(t *testing.T) {
	service1 := jwt.NewJWTService("secret-1", "refresh-1", time.Hour, time.Hour)
	service2 := jwt.NewJWTService("secret-2", "refresh-2", time.Hour, time.Hour)
	userID := uuid.New()

	pair, err := service1.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service2.ValidateAccessToken(pair.AccessToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
}

func TestJWTService_ValidateRefreshToken_Success(t *testing.T) {
	service := jwt.NewJWTService("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	claims, err := service.ValidateRefreshToken(pair.RefreshToken)
	assert.NoError(t, err)
	assert.Equal(t, userID.String(), claims.UserID)
	assert.Equal(t, jwt.TokenTypeRefresh, claims.TokenType)
}

func TestJWTService_RefreshTokens_Success(t *testing.T) {
	service := jwt.NewJWTService("access-secret", "refresh-secret", time.Hour, time.Hour)
	userID := uuid.New()

	pair, err := service.GenerateTokenPair(userID, "test@example.com", "Test User", entity.RoleAdmin)
	require.NoError(t, err)

	time.Sleep(1 * time.Second)

	newPair, err := service.RefreshTokens(pair.RefreshToken)
	assert.NoError(t, err)
	assert.NotEmpty(t, newPair.AccessToken)
	assert.NotEmpty(t, newPair.RefreshToken)
	assert.NotEqual(t, pair.AccessToken, newPair.AccessToken)
}
