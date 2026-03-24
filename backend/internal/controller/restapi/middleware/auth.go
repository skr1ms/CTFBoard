package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/crypto"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

type contextKey string

const UserRoleKey contextKey = "role"

type APITokenAuther interface {
	GetByTokenHash(ctx context.Context, tokenHash string) (*domain.APIToken, error)
	UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error
	ValidateToken(t *domain.APIToken) bool
}

type UserByIDGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

func authBearer(jwtService jwtkit.Service, r *http.Request, token string) (context.Context, bool) {
	claims, err := jwtService.ValidateAccessToken(r.Context(), token)
	if err != nil {
		return nil, false
	}
	ctx := jwtkit.ClaimsIntoContext(r.Context(), claims)
	if id, ok := jwtkit.UserIDFromContext(ctx); ok {
		ctx = context.WithValue(ctx, httputil.UserIDKey, id.String())
	}
	return ctx, true
}

func authAPIToken(apiTokenUC APITokenAuther, userUC UserByIDGetter, log logkit.Logger, r *http.Request, plaintext string) (context.Context, bool) {
	if apiTokenUC == nil || userUC == nil {
		return nil, false
	}
	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, false
	}
	tokenHash := crypto.SHA256Hex(plaintext)
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
	if user.WasInBannedTeam && user.Role != domain.RoleAdmin {
		return nil, false
	}
	if err := apiTokenUC.UpdateLastUsedAt(r.Context(), token.ID); err != nil {
		log.WithError(err).Warn("middleware - Auth - UpdateLastUsedAt: failed to update api token last_used_at")
	}
	ctx := context.WithValue(r.Context(), httputil.UserIDKey, user.ID.String())
	ctx = context.WithValue(ctx, UserRoleKey, string(user.Role))
	ctx = context.WithValue(ctx, userContextKey, user)
	return ctx, true
}

func Auth(jwtService jwtkit.Service, apiTokenUC APITokenAuther, userUC UserByIDGetter, log logkit.Logger) func(http.Handler) http.Handler {
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
				ctx, ok = authBearer(jwtService, r, jwtkit.ExtractRaw(r))
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
		if !ok || user == nil || user.Role != domain.RoleAdmin {
			httputil.HandleError(w, r, httperr.ErrAccessDenied)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserID(ctx context.Context) string {
	if id, ok := jwtkit.UserIDFromContext(ctx); ok {
		return id.String()
	}
	return httputil.GetUserID(ctx)
}

func GetUserRole(ctx context.Context) string {
	if role, ok := jwtkit.RoleFromContext(ctx); ok {
		return role
	}
	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}
