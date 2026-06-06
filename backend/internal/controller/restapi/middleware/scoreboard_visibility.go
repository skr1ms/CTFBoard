package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
)

const (
	scoreboardVisibilityKey = "scoreboard_visibility"
	scoreboardVisibilityTTL = 30 * time.Second
)

// ScoreboardSettingsGetter is the minimal interface required by ScoreboardVisibility middleware.
type ScoreboardSettingsGetter interface {
	Get(ctx context.Context) (*domain.Settings, error)
}

// ScoreboardVisibilityCache holds shared TTL cache for scoreboard visibility; use one instance app-wide and call Invalidate after PUT /admin/settings.
type ScoreboardVisibilityCache struct {
	cv *cachekit.CachedValue[string]
}

func NewScoreboardVisibilityCache(ctx context.Context) *ScoreboardVisibilityCache {
	if ctx == nil {
		panic("middleware.NewScoreboardVisibilityCache: nil context")
	}

	return &ScoreboardVisibilityCache{
		cv: cachekit.NewCachedValue[string](ctx, scoreboardVisibilityKey, scoreboardVisibilityTTL),
	}
}

func (s *ScoreboardVisibilityCache) Invalidate() {
	s.cv.Invalidate()
}

// Middleware returns an HTTP middleware that enforces the scoreboard_visible
// setting. Visibility is loaded via CachedValue (scoreboardVisibilityTTL=30s)
// and re-fetched only after expiry or explicit Invalidate. Admin users bypass
// the check. The three modes are: public (pass-through), hidden (403 for all),
// admins_only (403 for non-admins).
func (s *ScoreboardVisibilityCache) Middleware(settingsGetter ScoreboardSettingsGetter) func(http.Handler) http.Handler {
	load := func(ctx context.Context) (string, error) {
		settings, err := settingsGetter.Get(ctx)
		if err != nil {
			return "", err
		}

		return settings.ScoreboardVisible, nil
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isAdmin(r.Context()) {
				next.ServeHTTP(w, r)

				return
			}

			visibility, err := s.cv.Get(r.Context(), load)
			if err != nil {
				httputil.HandleError(w, r, errmap.MapAppError(err))

				return
			}

			switch visibility {
			case domain.ScoreboardVisiblePublic:
				next.ServeHTTP(w, r)
			case domain.ScoreboardVisibleHidden:
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrScoreboardHidden))

				return
			case domain.ScoreboardVisibleAdminsOnly:
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrScoreboardAdminsOnly))

				return
			default:
				httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrScoreboardAccessDenied))

				return
			}
		})
	}
}

// ScoreboardVisibility is a convenience constructor that creates a
// ScoreboardVisibilityCache and returns its Middleware. Use this when a
// shared cache instance is not needed (e.g. in tests or simple setups);
// for production use the shared ScoreboardVisibilityCache from wire/providers.
func ScoreboardVisibility(ctx context.Context, settingsGetter ScoreboardSettingsGetter) func(http.Handler) http.Handler {
	c := NewScoreboardVisibilityCache(ctx)

	return c.Middleware(settingsGetter)
}
