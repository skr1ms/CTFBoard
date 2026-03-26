package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
)

const teamBanCacheTTL = 50 * time.Millisecond

// RequireTeamNotBanned uses cache with teamBanCacheTTL. After BanTeam/UnbanTeam
// the usecase invalidates the team cache; a short TTL limits the window where a banned
// team could still be seen as active before invalidation or expiry.

type TeamGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Team, error)
}

func RequireTeamNotBanned(teamGetter TeamGetter, c *cachekit.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil || user.Role == domain.RoleAdmin || user.TeamID == nil {
				next.ServeHTTP(w, r)

				return
			}

			teamIDStr := user.TeamID.String()

			var (
				team *domain.Team
				err  error
			)

			if c != nil {
				team, err = cachekit.GetOrLoad(c, r.Context(), cache.KeyTeam(teamIDStr), teamBanCacheTTL, func(context.Context) (*domain.Team, error) {
					return teamGetter.GetByID(r.Context(), *user.TeamID)
				})
			} else {
				team, err = teamGetter.GetByID(r.Context(), *user.TeamID)
			}

			if err != nil {
				httputil.HandleError(w, r, err)

				return
			}

			if team == nil {
				next.ServeHTTP(w, r)

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
			if !ok || user == nil || user.Role == domain.RoleAdmin {
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
