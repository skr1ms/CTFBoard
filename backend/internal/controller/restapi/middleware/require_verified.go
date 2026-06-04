package middleware

import (
	"context"
	"net/http"

	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// VerificationSettingsGetter is the minimal interface required to read live
// email-verification policy from application settings.
type VerificationSettingsGetter interface {
	Get(ctx context.Context) (*domain.Settings, error)
}

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

// RequireVerifiedFromSettings enforces email verification from live app settings
// so setup/admin changes take effect without rebuilding the router. fallback is
// the env/provider capability gate and is also used if settings cannot be read.
func RequireVerifiedFromSettings(fallback bool, getter VerificationSettingsGetter, log logkit.Logger) func(http.Handler) http.Handler {
	if getter == nil {
		return RequireVerified(fallback)
	}

	if log == nil {
		log = logkit.Noop()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			verifyEmails := fallback
			settings, err := getter.Get(r.Context())
			if err != nil {
				log.WithError(err).Warn("middleware - RequireVerifiedFromSettings - Settings.Get: using fallback")
			} else if settings != nil {
				verifyEmails = fallback && settings.VerifyEmails
			}

			RequireVerified(verifyEmails)(next).ServeHTTP(w, r)
		})
	}
}
