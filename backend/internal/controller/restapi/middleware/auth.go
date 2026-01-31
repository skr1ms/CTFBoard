package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
	"github.com/skr1ms/CTFBoard/pkg/jwt"
)

type contextKey string

const UserRoleKey contextKey = "role"

func Auth(jwtService *jwt.JWTService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				httputil.RenderError(w, r, http.StatusUnauthorized, "authorization header required")
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				httputil.RenderError(w, r, http.StatusUnauthorized, "invalid authorization header format")
				return
			}

			claims, err := jwtService.ValidateAccessToken(parts[1])
			if err != nil {
				httputil.RenderError(w, r, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), httputil.UserIDKey, claims.UserID)
			ctx = context.WithValue(ctx, UserRoleKey, claims.Role)
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
