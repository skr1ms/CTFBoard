package middleware

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/usecase/user"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

type userContextKeyType string

const userContextKey userContextKeyType = entity.RoleUser

func InjectUser(userUC *user.UserUseCase) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := httputil.GetUserID(r.Context())
			if userID == "" {
				next.ServeHTTP(w, r)
				return
			}

			useruuid, err := uuid.Parse(userID)
			if err != nil {
				httputil.RenderError(w, r, http.StatusBadRequest, "invalid user ID")
				return
			}

			user, err := userUC.GetByID(r.Context(), useruuid)
			if err != nil {
				httputil.RenderError(w, r, http.StatusUnauthorized, "user not found")
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
