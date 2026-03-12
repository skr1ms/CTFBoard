package jwt

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// MinSecretLength is the minimum required byte length for JWT signing secrets.
const MinSecretLength = 32

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
	AccessMethod     = "HS256"
	RefreshMethod    = "HS256"
	IssuerAstroCTFb  = "astroctfb"
)

// UserRoleLookup resolves the current role for a user during token refresh.
type UserRoleLookup func(ctx context.Context, userID uuid.UUID) (email, name, role string, err error)

// KeyEntry holds a key id and secret for signing/verification (key rotation).
type KeyEntry struct {
	Kid    string
	Secret string
}

type Service interface {
	GenerateTokenPair(userID uuid.UUID, email, name, role string) (*TokenPair, error)
	ValidateAccessToken(ctx context.Context, tokenString string) (*CustomClaims, error)
	ValidateRefreshToken(ctx context.Context, tokenString string) (*CustomClaims, error)
	RefreshTokens(ctx context.Context, refreshTokenString string) (*TokenPair, error)
	RevokeRefreshToken(ctx context.Context, refreshTokenString string) error
	RevokeAccessToken(ctx context.Context, accessTokenString string) error
	// RevokeAllForUser invalidates every token issued to userID before the
	// current moment. Tokens with IssuedAt <= the recorded timestamp are
	// rejected by subsequent ValidateAccessToken / ValidateRefreshToken calls.
	RevokeAllForUser(ctx context.Context, userID uuid.UUID) error
}

type JWTService struct {
	accessKeysByKid   map[string][]byte
	refreshKeysByKid  map[string][]byte
	accessPrimaryKid  string
	refreshPrimaryKid string
	accessTTL         time.Duration
	refreshTTL        time.Duration
	revoker           RevocationStore
	userRoleLookup    atomic.Pointer[UserRoleLookup]
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
	accessKeys []KeyEntry,
	refreshKeys []KeyEntry,
	accessTTL time.Duration,
	refreshTTL time.Duration,
	revoker RevocationStore,
	userRoleLookup UserRoleLookup,
) (*JWTService, error) {
	if len(accessKeys) == 0 {
		return nil, fmt.Errorf("access keys must contain at least one key")
	}
	if len(refreshKeys) == 0 {
		return nil, fmt.Errorf("refresh keys must contain at least one key")
	}
	accessByKid := make(map[string][]byte, len(accessKeys))
	for _, k := range accessKeys {
		if len(k.Secret) < MinSecretLength {
			return nil, fmt.Errorf("access key %q secret must be at least %d bytes", k.Kid, MinSecretLength)
		}
		accessByKid[k.Kid] = []byte(k.Secret)
	}
	refreshByKid := make(map[string][]byte, len(refreshKeys))
	for _, k := range refreshKeys {
		if len(k.Secret) < MinSecretLength {
			return nil, fmt.Errorf("refresh key %q secret must be at least %d bytes", k.Kid, MinSecretLength)
		}
		refreshByKid[k.Kid] = []byte(k.Secret)
	}
	svc := &JWTService{
		accessKeysByKid:   accessByKid,
		refreshKeysByKid:  refreshByKid,
		accessPrimaryKid:  accessKeys[0].Kid,
		refreshPrimaryKid: refreshKeys[0].Kid,
		accessTTL:         accessTTL,
		refreshTTL:        refreshTTL,
		revoker:           revoker,
	}
	if userRoleLookup != nil {
		svc.userRoleLookup.Store(&userRoleLookup)
	}
	return svc, nil
}

func (j *JWTService) SetUserRoleLookup(fn UserRoleLookup) {
	j.userRoleLookup.Store(&fn)
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
			Issuer:    IssuerAstroCTFb,
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
			Issuer:    IssuerAstroCTFb,
		},
	}

	accessToken := jwt.NewWithClaims(jwt.GetSigningMethod(AccessMethod), accessClaims)
	accessToken.Header["kid"] = j.accessPrimaryKid
	accessKey := j.accessKeysByKid[j.accessPrimaryKid]
	accessTokenString, err := accessToken.SignedString(accessKey)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken := jwt.NewWithClaims(jwt.GetSigningMethod(RefreshMethod), refreshClaims)
	refreshToken.Header["kid"] = j.refreshPrimaryKid
	refreshKey := j.refreshKeysByKid[j.refreshPrimaryKid]
	refreshTokenString, err := refreshToken.SignedString(refreshKey)
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

func (j *JWTService) ValidateAccessToken(ctx context.Context, tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		var kid string
		if k, ok := token.Header["kid"].(string); ok {
			kid = k
		}
		if kid == "" {
			kid = j.accessPrimaryKid
		}
		key, ok := j.accessKeysByKid[kid]
		if !ok {
			return nil, fmt.Errorf("unknown access key id %q", kid)
		}
		return key, nil
	}, jwt.WithIssuer(IssuerAstroCTFb))
	if err != nil {
		return nil, fmt.Errorf("failed to validate access token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, fmt.Errorf("invalid token type")
	}
	if err := j.checkRevocation(ctx, claims); err != nil {
		return nil, fmt.Errorf("jwt validate access: %w", err)
	}
	return claims, nil
}

func (j *JWTService) ValidateRefreshToken(ctx context.Context, tokenString string) (*CustomClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		var kid string
		if k, ok := token.Header["kid"].(string); ok {
			kid = k
		}
		if kid == "" {
			kid = j.refreshPrimaryKid
		}
		key, ok := j.refreshKeysByKid[kid]
		if !ok {
			return nil, fmt.Errorf("unknown refresh key id %q", kid)
		}
		return key, nil
	}, jwt.WithIssuer(IssuerAstroCTFb))
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid refresh token")
	}
	if claims.TokenType != TokenTypeRefresh {
		return nil, fmt.Errorf("invalid token type")
	}
	if err := j.checkRevocation(ctx, claims); err != nil {
		return nil, fmt.Errorf("jwt validate refresh: %w", err)
	}
	return claims, nil
}

// checkRevocation checks both the per-JTI revocation list and the per-user
// revocation timestamp. It returns an error if the token is revoked by either.
func (j *JWTService) checkRevocation(ctx context.Context, claims *CustomClaims) error {
	if claims.ID == "" {
		return fmt.Errorf("token missing jti claim")
	}
	if j.revoker == nil {
		return nil
	}
	revoked, err := j.revoker.IsRevoked(ctx, claims.ID)
	if err != nil {
		return fmt.Errorf("revocation check: %w", err)
	}
	if revoked {
		return fmt.Errorf("token revoked")
	}
	// Check per-user revocation (e.g. set on password change).
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return fmt.Errorf("invalid user_id in claims: %w", err)
	}
	var issuedAt int64
	if claims.IssuedAt != nil {
		issuedAt = claims.IssuedAt.Unix()
	}
	userRevoked, err := j.revoker.IsUserRevoked(ctx, userID, issuedAt)
	if err != nil {
		return fmt.Errorf("user revocation check: %w", err)
	}
	if userRevoked {
		return fmt.Errorf("token revoked")
	}
	return nil
}

func (j *JWTService) RevokeRefreshToken(ctx context.Context, refreshTokenString string) error {
	claims, err := j.ValidateRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return fmt.Errorf("jwt revoke: %w", err)
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

func (j *JWTService) RefreshTokens(ctx context.Context, refreshTokenString string) (*TokenPair, error) {
	claims, err := j.ValidateRefreshToken(ctx, refreshTokenString)
	if err != nil {
		return nil, fmt.Errorf("failed to validate refresh token: %w", err)
	}
	if j.revoker != nil && claims.ID != "" {
		ttl := j.refreshTTL
		if claims.ExpiresAt != nil {
			ttl = time.Until(claims.ExpiresAt.Time)
			if ttl <= 0 {
				ttl = j.refreshTTL
			}
		}
		if err := j.revoker.Revoke(ctx, claims.ID, ttl); err != nil {
			return nil, fmt.Errorf("failed to revoke old refresh token: %w", err)
		}
	}
	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token claims: %w", err)
	}

	email, name, role := claims.Email, claims.FullName, claims.Role
	if fn := j.userRoleLookup.Load(); fn != nil {
		freshEmail, freshName, freshRole, lookupErr := (*fn)(ctx, userID)
		if lookupErr != nil {
			return nil, fmt.Errorf("failed to lookup user role during refresh: %w", lookupErr)
		}
		email, name, role = freshEmail, freshName, freshRole
	}

	return j.GenerateTokenPair(userID, email, name, role)
}

func (j *JWTService) RevokeAccessToken(ctx context.Context, accessTokenString string) error {
	claims, err := j.ValidateAccessToken(ctx, accessTokenString)
	if err != nil {
		// The token is already invalid (expired, malformed, or revoked), so there
		// is nothing to revoke. Return nil to let logout succeed unconditionally.
		return nil
	}
	if claims.ID == "" || j.revoker == nil {
		return nil
	}
	ttl := j.accessTTL
	if claims.ExpiresAt != nil {
		ttl = time.Until(claims.ExpiresAt.Time)
		if ttl <= 0 {
			return nil
		}
	}
	return j.revoker.Revoke(ctx, claims.ID, ttl)
}

// RevokeAllForUser invalidates all tokens currently issued to userID by storing
// a user-level revocation timestamp. Any token whose IssuedAt is <= that
// timestamp will be rejected by ValidateAccessToken / ValidateRefreshToken.
// Uses the longer of the two TTLs so the entry outlives all valid tokens.
func (j *JWTService) RevokeAllForUser(ctx context.Context, userID uuid.UUID) error {
	if j.revoker == nil {
		return nil
	}
	ttl := j.refreshTTL
	if j.accessTTL > ttl {
		ttl = j.accessTTL
	}
	return j.revoker.RevokeUserTokens(ctx, userID, ttl)
}
