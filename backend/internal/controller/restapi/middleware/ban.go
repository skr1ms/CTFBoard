package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/google/uuid"
)

const teamBanCacheTTL = 15 * time.Second

type TeamGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*entity.Team, error)
}

func RequireTeamNotBanned(teamGetter TeamGetter, c *cache.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil || user.Role == entity.RoleAdmin || user.TeamID == nil {
				next.ServeHTTP(w, r)
				return
			}

			teamIDStr := user.TeamID.String()
			var team *entity.Team
			var err error
			if c != nil {
				team, err = cache.GetOrLoad(c, r.Context(), cache.KeyTeam(teamIDStr), teamBanCacheTTL, func() (*entity.Team, error) {
					return teamGetter.GetByID(r.Context(), *user.TeamID)
				})
			} else {
				team, err = teamGetter.GetByID(r.Context(), *user.TeamID)
			}
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}

			if team.IsBanned {
				httputil.HandleError(w, r, httperr.ErrTeamBanned)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func RequireUserNotBanned() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil || user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			if user.IsBanned {
				httputil.HandleError(w, r, httperr.ErrUserBanned)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
