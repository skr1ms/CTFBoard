package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/jwt"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
)

type contextKey string

const UserRoleKey contextKey = "role"

type APITokenAuther interface {
	GetByTokenHash(ctx context.Context, tokenHash string) (*entity.APIToken, error)
	UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error
	ValidateToken(t *entity.APIToken) bool
}

type UserByIDGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
}

func authBearer(jwtService jwt.Service, r *http.Request, token string) (context.Context, bool) {
	claims, err := jwtService.ValidateAccessToken(r.Context(), token)
	if err != nil {
		return nil, false
	}
	ctx := context.WithValue(r.Context(), httputil.UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
	return ctx, true
}

func authAPIToken(apiTokenUC APITokenAuther, userUC UserByIDGetter, log logger.Logger, r *http.Request, plaintext string) (context.Context, bool) {
	if apiTokenUC == nil || userUC == nil {
		return nil, false
	}
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, false
	}
	hash := sha256.Sum256([]byte(plaintext))
	tokenHash := hex.EncodeToString(hash[:])
	token, err := apiTokenUC.GetByTokenHash(r.Context(), tokenHash)
	if err != nil || token == nil || !apiTokenUC.ValidateToken(token) {
		return nil, false
	}
	user, err := userUC.GetByID(r.Context(), token.UserID)
	if err != nil || user == nil {
		return nil, false
	}
	if user.IsBanned {
		return nil, false
	}
	if err := apiTokenUC.UpdateLastUsedAt(r.Context(), token.ID); err != nil {
		log.WithError(err).Warn("middleware - Auth - UpdateLastUsedAt: failed to update api token last_used_at")
	}
	ctx := context.WithValue(r.Context(), httputil.UserIDKey, user.ID.String())
	ctx = context.WithValue(ctx, UserRoleKey, user.Role)
	ctx = context.WithValue(ctx, userContextKey, user)
	return ctx, true
}

func Auth(jwtService jwt.Service, apiTokenUC APITokenAuther, userUC UserByIDGetter, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httputil.HandleError(w, r, httperr.ErrAuthorizationHeaderRequired)
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				httputil.HandleError(w, r, httperr.ErrInvalidAuthorizationHeader)
				return
			}
			var ctx context.Context
			var ok bool
			switch {
			case strings.EqualFold(parts[0], "Bearer"):
				ctx, ok = authBearer(jwtService, r, parts[1])
			case strings.EqualFold(parts[0], "Token"):
				ctx, ok = authAPIToken(apiTokenUC, userUC, log, r, parts[1])
			default:
				httputil.HandleError(w, r, httperr.ErrInvalidAuthorizationHeader)
				return
			}
			if !ok {
				httputil.HandleError(w, r, httperr.ErrInvalidToken)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r.Context())
		if !ok || user == nil || user.Role != entity.RoleAdmin {
			httputil.HandleError(w, r, httperr.ErrAccessDenied)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) string {
	return httputil.GetUserID(ctx)
}

func GetUserRole(ctx context.Context) string {
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}
