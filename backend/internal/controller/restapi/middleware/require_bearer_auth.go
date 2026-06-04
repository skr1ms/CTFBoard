package middleware

import (
	"net/http"

	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-jwtkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
)

// RequireBearerAuth rejects requests authenticated only via long-lived API tokens.
// Token-management endpoints need a fresh browser/JWT session so a leaked API
// token cannot mint or hide replacement credentials.
func RequireBearerAuth() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, ok := jwtkit.UserIDFromContext(r.Context()); !ok {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrAccessDenied))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
