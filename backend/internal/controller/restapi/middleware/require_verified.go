package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// RequireVerified returns a middleware that rejects unverified users when verifyEmails is true.
// When verifyEmails is false, the middleware is a no-op pass-through.
func RequireVerified(verifyEmails bool) func(http.Handler) http.Handler {
	if !verifyEmails {
		return func(next http.Handler) http.Handler { return next }
	}

	return RequireAuth(func(user *domain.User) error {
		if !user.IsVerified {
			return apperr.ErrEmailNotVerified
		}

		return nil
	})
}
