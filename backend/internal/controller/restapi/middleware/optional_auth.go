package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// OptionalAuth authenticates the request when credentials are present but passes
// through silently when no Authorization header is provided or credentials are invalid.
// On success, identity is injected into context identically to Auth.
func OptionalAuth(jwtService jwtkit.Service, apiTokenUC APITokenAuther, userUC UserByIDGetter, log logkit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)

				return
			}

			parts := strings.SplitN(authHeader, " ", authHeaderPartCount)
			if len(parts) != authHeaderPartCount {
				next.ServeHTTP(w, r)

				return
			}

			var (
				ctx context.Context
				ok  bool
			)

			switch {
			case strings.EqualFold(parts[0], "Bearer"):
				ctx, ok = authBearer(jwtService, r, parts[1])
			case strings.EqualFold(parts[0], "Token"):
				ctx, ok = authAPIToken(apiTokenUC, userUC, log, r, parts[1])
			}

			if ok {
				next.ServeHTTP(w, r.WithContext(ctx)) //nolint:contextcheck // ctx is derived from r.Context by authBearer/authAPIToken and carries authenticated identity.
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}

// OptionalInjectUser loads the authenticated user into context when a user ID is present.
// Unlike InjectUser, it passes through without error when no user ID is found (unauthenticated request).
func OptionalInjectUser(userUC UserByIDGetter, _ *cachekit.Cache, log logkit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				next.ServeHTTP(w, r)

				return
			}

			if _, already := r.Context().Value(userContextKey).(*domain.User); already {
				next.ServeHTTP(w, r)

				return
			}

			userUUID, err := uuid.Parse(userID)
			if err != nil {
				next.ServeHTTP(w, r)

				return
			}

			user, err := userUC.GetByID(r.Context(), userUUID)
			if err != nil || user == nil {
				next.ServeHTTP(w, r)

				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
