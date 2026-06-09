package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

type userContextKeyType = contextKey

const (
	userContextKey userContextKeyType = "authenticated_user"
)

// InjectUser is a middleware that loads the authenticated user by ID and stores it in the
// request context under userContextKey. Authentication and authorization decisions depend
// on mutable fields such as team_id, role, and ban flags, so this middleware reads the
// current user record on every request instead of serving a cached snapshot.
// Must run after the Auth middleware which sets the user ID in the context.
func InjectUser(userUC UserByIDGetter, _ *cachekit.Cache, log logkit.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := GetUserID(r.Context())
			if userID == "" {
				if log != nil {
					log.Warn("InjectUser - userID is empty (check middleware order: Auth before InjectUser)")
				}

				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrNotAuthenticated))

				return
			}

			if _, already := r.Context().Value(userContextKey).(*domain.User); already {
				next.ServeHTTP(w, r)

				return
			}

			userUUID, err := uuid.Parse(userID)
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.NewValidationErrorf("invalid user ID")))

				return
			}

			user, err := userUC.GetByID(r.Context(), userUUID)
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			if user == nil {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrUserNotFound))

				return
			}

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUser retrieves the authenticated *domain.User stored in ctx by InjectUser or
// authAPIToken. Returns (nil, false) when no user is present (unauthenticated request
// or middleware not in the chain).
func GetUser(ctx context.Context) (*domain.User, bool) {
	user, ok := ctx.Value(userContextKey).(*domain.User)

	return user, ok
}

func isAdmin(ctx context.Context) bool {
	user, ok := GetUser(ctx)

	return ok && user != nil && user.Role == domain.RoleAdmin
}
