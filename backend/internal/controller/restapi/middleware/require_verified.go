package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func RequireVerified(verifyEmails bool) func(http.Handler) http.Handler {
	if !verifyEmails {
		return func(next http.Handler) http.Handler { return next }
	}

	return RequireAuth(func(user *domain.User) error {
		if !user.IsVerified {
			return httperr.ErrEmailNotVerified
		}

		return nil
	})
}
