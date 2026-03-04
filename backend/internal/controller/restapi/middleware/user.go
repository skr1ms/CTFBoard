package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/google/uuid"
)

type userContextKeyType = contextKey

const (
	userContextKey userContextKeyType = "authenticated_user"
	userCacheTTL                      = 30 * time.Second
)

//nolint:gocognit // middleware: auth check + cache branch + parse + load
func InjectUser(userUC usecase.UserUseCase, c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := httputil.GetUserID(r.Context())
			if userID == "" {
				next.ServeHTTP(w, r)
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

			ctx := context.WithValue(r.Context(), userContextKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUser(ctx context.Context) (*entity.User, bool) {
	user, ok := ctx.Value(userContextKey).(*entity.User)
	return user, ok
}
