package jwt

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	AccessMethod     = "HS256"
	RefreshMethod    = "HS256"
	IssuerCTFBoard   = "ctfboard"
)

type Service interface {
	GenerateTokenPair(userID uuid.UUID, email, name, role string) (*TokenPair, error)
	ValidateAccessToken(tokenString string) (*CustomClaims, error)
	ValidateRefreshToken(tokenString string) (*CustomClaims, error)
	RefreshTokens(refreshTokenString string) (*TokenPair, error)
	RevokeRefreshToken(ctx context.Context, refreshTokenString string) error
}

type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	revoker       RevocationStore
}

type CustomClaims struct {
	UserID    string `json:"user_id"`
	Email     string `json:"email"`
	FullName  string `json:"full_name"`
	Role      string `json:"role,omitempty"`
	TokenType string `json:"token_type"`
	jwt.RegisteredClaims
}

type TokenPair struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	AccessExpiresAt  int64  `json:"access_expires_at"`
	RefreshExpiresAt int64  `json:"refresh_expires_at"`
}

func NewJWTService(
	accessSecret string,
	refreshSecret string,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	revoker RevocationStore,
) *JWTService {
	return &JWTService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
		revoker:       revoker,
	}
}

func (j *JWTService) GenerateTokenPair(userID uuid.UUID, email, name, role string) (*TokenPair, error) {
	now := time.Now()
	accessExpiry := now.Add(j.accessTTL)
	refreshExpiry := now.Add(j.refreshTTL)

	accessJTI := uuid.New().String()
	refreshJTI := uuid.New().String()

	accessClaims := &CustomClaims{
		UserID:    userID.String(),
		Email:     email,
		FullName:  name,
		Role:      role,
		TokenType: TokenTypeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessJTI,
			ExpiresAt: jwt.NewNumericDate(accessExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    IssuerCTFBoard,
		},
	}

	refreshClaims := &CustomClaims{
		UserID:    userID.String(),
		Email:     email,
		FullName:  name,
		Role:      role,
		TokenType: TokenTypeRefresh,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        refreshJTI,
			ExpiresAt: jwt.NewNumericDate(refreshExpiry),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    IssuerCTFBoard,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.GetSigningMethod(AccessMethod), accessClaims)
	accessTokenString, err := accessToken.SignedString(j.accessSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.GetSigningMethod(RefreshMethod), refreshClaims)
	refreshTokenString, err := refreshToken.SignedString(j.refreshSecret)
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessTokenString,
		RefreshToken:     refreshTokenString,
		AccessExpiresAt:  accessExpiry.Unix(),
		RefreshExpiresAt: refreshExpiry.Unix(),
	}, nil
}

func (j *JWTService) ValidateAccessToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.accessSecret, nil
	}, jwt.WithIssuer(IssuerCTFBoard))
	if err != nil {
		return nil, fmt.Errorf("failed to validate access token: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.TokenType != TokenTypeAccess {
			return nil, fmt.Errorf("invalid token type")
		}
		if j.revoker != nil && claims.ID != "" {
			revoked, err := j.revoker.IsRevoked(context.Background(), claims.ID)
			if err != nil {
				return nil, fmt.Errorf("revocation check: %w", err)
			}
			if revoked {
				return nil, fmt.Errorf("token revoked")
			}
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func (j *JWTService) ValidateRefreshToken(tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return j.refreshSecret, nil
	}, jwt.WithIssuer(IssuerCTFBoard))
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	if claims, ok := token.Claims.(*CustomClaims); ok && token.Valid {
		if claims.TokenType != TokenTypeRefresh {
			return nil, fmt.Errorf("invalid token type")
		}
		if j.revoker != nil && claims.ID != "" {
			revoked, err := j.revoker.IsRevoked(context.Background(), claims.ID)
			if err != nil {
				return nil, fmt.Errorf("revocation check: %w", err)
			}
			if revoked {
				return nil, fmt.Errorf("token revoked")
			}
		}
		return claims, nil
	}

	return nil, fmt.Errorf("invalid refresh token")
}

func (j *JWTService) RevokeRefreshToken(ctx context.Context, refreshTokenString string) error {
	claims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return err
	}
	if claims.ID == "" {
		return nil
	}
	if j.revoker == nil {
		return nil
	}
	exp := claims.ExpiresAt
	var ttl time.Duration
	if exp != nil {
		ttl = time.Until(exp.Time)
	}
	if ttl <= 0 {
		ttl = j.refreshTTL
	}
	return j.revoker.Revoke(ctx, claims.ID, ttl)
}

func (j *JWTService) RefreshTokens(refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateRefreshToken(refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token claims: %w", err)
	}

	return j.GenerateTokenPair(userID, claims.Email, claims.FullName, claims.Role)
}
