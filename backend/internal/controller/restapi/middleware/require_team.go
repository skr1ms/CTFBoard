package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

// RequireTeam returns a middleware that rejects requests from authenticated users who are not in a team.
func RequireTeam() func(http.Handler) http.Handler {
	return RequireAuth(func(user *domain.User) error {
		if user.TeamID == nil {
			return apperr.ErrUserNotInTeam
		}

		return nil
	})
}
