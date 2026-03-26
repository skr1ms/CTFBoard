package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/wahrwelt-kit/go-cachekit"
	"github.com/wahrwelt-kit/go-httpkit/httputil"

	"github.com/TakuyaYagam1/AstroCTFb/internal/domain"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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

func NewScoreboardVisibilityCache() *ScoreboardVisibilityCache {
	return &ScoreboardVisibilityCache{
		cv: cachekit.NewCachedValue[string](context.Background(), scoreboardVisibilityKey, scoreboardVisibilityTTL),
	}
}

func (s *ScoreboardVisibilityCache) Invalidate() {
	s.cv.Invalidate()
}

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
			user, ok := GetUser(r.Context())
			if ok && user != nil && user.Role == domain.RoleAdmin {
				next.ServeHTTP(w, r)

				return
			}

			visibility, err := s.cv.Get(r.Context(), load)
			if err != nil {
				httputil.HandleError(w, r, err)

				return
			}

			switch visibility {
			case domain.ScoreboardVisiblePublic:
				next.ServeHTTP(w, r)
			case domain.ScoreboardVisibleHidden:
				httputil.HandleError(w, r, httperr.ErrScoreboardHidden)

				return
			case domain.ScoreboardVisibleAdminsOnly:
				httputil.HandleError(w, r, httperr.ErrScoreboardAdminsOnly)

				return
			default:
				httputil.HandleError(w, r, httperr.ErrScoreboardAccessDenied)

				return
			}
		})
	}
}

func ScoreboardVisibility(settingsGetter ScoreboardSettingsGetter) func(http.Handler) http.Handler {
	c := NewScoreboardVisibilityCache()

	return c.Middleware(settingsGetter)
}
