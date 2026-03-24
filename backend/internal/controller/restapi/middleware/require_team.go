package middleware

import (
	"net/http"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

func RequireTeam() func(http.Handler) http.Handler {
	return RequireAuth(func(user *domain.User) error {
		if user.TeamID == nil {
			return httperr.ErrUserNotInTeam
		}
		return nil
	})
}
