package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/cache"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const teamBanCacheTTL = 50 * time.Millisecond

// RequireTeamNotBanned uses cache with teamBanCacheTTL. After BanTeam/UnbanTeam
// the usecase invalidates the team cache; a short TTL limits the window where a banned
// team could still be seen as active before invalidation or expiry

type TeamGetter interface {
	GetByID(ctx context.Context, id uuid.UUID) (*domain.Team, error)
}

// RequireTeamNotBanned blocks requests from users whose team is banned.
// Team data is fetched via a short TTL cache (teamBanCacheTTL) to reduce database
// load while keeping the ban enforcement window small. Admin users and users
// without a team are passed through unconditionally.
func RequireTeamNotBanned(teamGetter TeamGetter, c *cachekit.Cache) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil || isAdmin(r.Context()) || user.TeamID == nil {
				next.ServeHTTP(w, r)

				return
			}

			teamIDStr := user.TeamID.String()

			var (
				team *domain.Team
				err  error
			)

			if c != nil {
				team, err = cachekit.GetOrLoad(c, r.Context(), cache.KeyTeam(teamIDStr), teamBanCacheTTL, func(ctx context.Context) (*domain.Team, error) {
					return teamGetter.GetByID(ctx, *user.TeamID)
				})
			} else {
				team, err = teamGetter.GetByID(r.Context(), *user.TeamID)
			}

			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			if team == nil {
				next.ServeHTTP(w, r)

				return
			}

			if team.IsBanned {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrTeamBanned))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireUserNotBanned blocks requests from users with direct or inherited ban state.
// Admin users and unauthenticated requests pass through unconditionally.
func RequireUserNotBanned() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if !ok || user == nil || isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			if user.IsBanned {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrUserBanned))

				return
			}

			if user.WasInBannedTeam {
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrUserWasInBannedTeam))

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
