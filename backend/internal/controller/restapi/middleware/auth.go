package middleware

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
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

func authBearer(jwtService *jwt.JWTService, r *http.Request, token string) (context.Context, bool) {
	claims, err := jwtService.ValidateAccessToken(token)
	if err != nil {
		return nil, false
	}
	ctx := context.WithValue(r.Context(), httputil.UserIDKey, claims.UserID)
	ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
	return ctx, true
}

func authAPIToken(apiTokenUC APITokenAuther, userUC UserByIDGetter, r *http.Request, plaintext string) (context.Context, bool) {
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
	_ = apiTokenUC.UpdateLastUsedAt(r.Context(), token.ID) //nolint:errcheck // best-effort update
	ctx := context.WithValue(r.Context(), httputil.UserIDKey, user.ID.String())
	ctx = context.WithValue(ctx, UserRoleKey, user.Role)
	return ctx, true
}

func Auth(jwtService *jwt.JWTService, apiTokenUC APITokenAuther, userUC UserByIDGetter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httputil.RenderError(w, r, http.StatusUnauthorized, "authorization header required")
				return
			}
			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 {
				httputil.RenderError(w, r, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
			var ctx context.Context
			var ok bool
			switch parts[0] {
			case "Bearer":
				ctx, ok = authBearer(jwtService, r, parts[1])
			case "Token":
				ctx, ok = authAPIToken(apiTokenUC, userUC, r, parts[1])
			default:
				httputil.RenderError(w, r, http.StatusUnauthorized, "invalid authorization header format")
				return
			}
			if !ok {
				httputil.RenderError(w, r, http.StatusUnauthorized, "invalid token")
				return
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		role, ok := r.Context().Value(UserRoleKey).(string)
		if !ok || role != entity.RoleAdmin {
			httputil.RenderError(w, r, http.StatusForbidden, "admin access required")
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
