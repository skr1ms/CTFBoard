package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

type userContextKeyType = contextKey

const (
	userContextKey userContextKeyType = "authenticated_user"
	userCacheTTL                      = 1 * time.Second
)

// InjectUser loads the user via cache (userCacheTTL) or usecase. After BanUser the
// usecase invalidates the user cache; a brief window until invalidation propagates is accepted.

func InjectUser(userUC usecase.UserUseCase, c *cache.Cache, log logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := httputil.GetUserID(r.Context())
			if userID == "" {
				if log != nil {
					log.Warn("InjectUser - userID is empty (check middleware order: Auth before InjectUser)")
				}
				httputil.HandleError(w, r, httperr.ErrNotAuthenticated)
				return
			}

			if _, already := r.Context().Value(userContextKey).(*entity.User); already {
				next.ServeHTTP(w, r)
				return
			}

			userUUID, err := uuid.Parse(userID)
			if err != nil {
				httputil.HandleError(w, r, httperr.NewValidationErrorf("invalid user ID"))
				return
			}

			var user *entity.User
			if c != nil {
				user, err = cache.GetOrLoad(c, r.Context(), cache.KeyUser(userID), userCacheTTL, func() (*entity.User, error) {
					return userUC.GetByID(r.Context(), userUUID)
				})
			} else {
				user, err = userUC.GetByID(r.Context(), userUUID)
			}
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}
			if user == nil {
				httputil.HandleError(w, r, httperr.ErrUserNotFound)
				return
			}
			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUser(ctx context.Context) (*entity.User, bool) {
	user, ok := ctx.Value(userContextKey).(*entity.User)
	return user, ok
}
