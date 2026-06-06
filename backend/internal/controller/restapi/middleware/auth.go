package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type contextKey string

// UserRoleKey is the context key under which the authenticated user's role string is stored by Auth middleware.
const UserRoleKey contextKey = "role"

const (
	apiTokenLastUsedTimeout = 2 * time.Second
	authHeaderPartCount     = 2
)

// APITokenAuther is the minimal interface required to validate API token credentials.
type APITokenAuther interface {
	AuthenticatePlaintext(ctx context.Context, plaintext string) (*domain.APIToken, error)
	UpdateLastUsedAt(ctx context.Context, id uuid.UUID) error
}

// UserByIDGetter is the minimal interface required to load a user by ID during API token authentication.
type UserByIDGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error)
}

// authBearer validates a JWT Bearer token and, on success, injects JWT claims
// and the user ID string into the context (via jwtkit and httputil.UserIDKey).
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

// authAPIToken authenticates a plain API token (Token scheme). Steps:
// SHA-256 hash the plaintext -> load token record -> validate expiry ->
// load user -> check ban flags -> update last_used_at asynchronously (best-effort) ->
// build context with userID, role, and full user object for downstream middleware.
func authAPIToken(apiTokenUC APITokenAuther, userUC UserByIDGetter, log logkit.Logger, r *http.Request, plaintext string) (context.Context, bool) {
	if apiTokenUC == nil || userUC == nil {
		return nil, false
	}

	plaintext = strings.TrimSpace(plaintext)
	if plaintext == "" {
		return nil, false
	}

	token, err := apiTokenUC.AuthenticatePlaintext(r.Context(), plaintext)
	if err != nil || token == nil {
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

	go func() {
		ctx, cancel := context.WithTimeout(context.WithoutCancel(r.Context()), apiTokenLastUsedTimeout)
		defer cancel()

		if err := apiTokenUC.UpdateLastUsedAt(ctx, token.ID); err != nil {
			log.WithError(err).Warn("middleware - Auth - UpdateLastUsedAt: failed to update api token last_used_at")
		}
	}()

	ctx := context.WithValue(r.Context(), httputil.UserIDKey, user.ID.String())
	ctx = context.WithValue(ctx, UserRoleKey, string(user.Role))
	ctx = context.WithValue(ctx, userContextKey, user)

	return ctx, true
}

// Auth authenticates requests using either a JWT Bearer token or a plain API token (Token scheme).
// On success, user identity is injected into the context for downstream middleware.
func Auth(jwtService jwtkit.Service, apiTokenUC APITokenAuther, userUC UserByIDGetter, log logkit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")

			// Browser WebSocket API cannot set custom headers during the HTTP->WS
			// upgrade, so the JWT is passed via Sec-WebSocket-Protocol: bearer, <token>.
			// Extract the non-"bearer" part of the subprotocol list as the credential.
			if authHeader == "" {
				if wsProto := r.Header.Get("Sec-WebSocket-Protocol"); wsProto != "" {
					for part := range strings.SplitSeq(wsProto, ",") {
						part = strings.TrimSpace(part)
						if part != "" && !strings.EqualFold(part, "bearer") {
							authHeader = "Bearer " + part

							break
						}
					}
				}
			}

			if authHeader == "" {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAuthorizationHeaderRequired))

				return
			}

			parts := strings.SplitN(authHeader, " ", authHeaderPartCount)
			if len(parts) != authHeaderPartCount {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrInvalidAuthorizationHeader))

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
			default:
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrInvalidAuthorizationHeader))

				return
			}

			if !ok {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrInvalidToken))

				return
			}

			next.ServeHTTP(w, r.WithContext(ctx)) //nolint:contextcheck // ctx is derived from r.Context by authBearer/authAPIToken and carries authenticated identity.
		})
	}
}

// Admin rejects requests from non-admin users with 403 Access Denied.
func Admin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, ok := GetUser(r.Context())
		if !ok || user == nil || user.Role != domain.RoleAdmin {
			httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))

			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetUserID returns the authenticated user ID from context, checking JWT claims first then the generic key.
func GetUserID(ctx context.Context) string {
	if id, ok := jwtkit.UserIDFromContext(ctx); ok {
		return id.String()
	}

	return httputil.GetUserID(ctx)
}

// GetUserRole returns the authenticated user's role string from context, checking JWT claims first.
func GetUserRole(ctx context.Context) string {
	if role, ok := jwtkit.RoleFromContext(ctx); ok {
		return role
	}

	if role, ok := ctx.Value(UserRoleKey).(string); ok {
		return role
	}

	return ""
}
