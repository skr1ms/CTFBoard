package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/entity"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/cache"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"golang.org/x/sync/singleflight"
)

const (
	scoreboardVisibilityKey = "scoreboard_visibility"
	scoreboardVisibilityTTL = 30 * time.Second
)

// ScoreboardSettingsGetter is the minimal interface required by ScoreboardVisibility middleware.
type ScoreboardSettingsGetter interface {
	Get(ctx context.Context) (*entity.Settings, error)
}

func ScoreboardVisibility(settingsGetter ScoreboardSettingsGetter) func(http.Handler) http.Handler {
	c := cache.NewTTLCache[string, string](scoreboardVisibilityTTL, 1)
	var sf singleflight.Group

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if ok && user != nil && user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			visibility, err := getScoreboardVisibility(r.Context(), c, &sf, settingsGetter)
			if err != nil {
				httputil.HandleError(w, r, err)
				return
			}

			switch visibility {
			case entity.ScoreboardVisiblePublic:
				next.ServeHTTP(w, r)
			case entity.ScoreboardVisibleHidden:
				httputil.HandleError(w, r, httperr.ErrScoreboardHidden)
				return
			case entity.ScoreboardVisibleAdminsOnly:
				httputil.HandleError(w, r, httperr.ErrScoreboardAdminsOnly)
				return
			default:
				httputil.HandleError(w, r, httperr.ErrScoreboardAccessDenied)
				return
			}
		})
	}
}

func getScoreboardVisibility(ctx context.Context, c *cache.TTLCache[string, string], sf *singleflight.Group, getter ScoreboardSettingsGetter) (string, error) {
	if v, ok := c.Get(scoreboardVisibilityKey); ok {
		return v, nil
	}
	v, err, _ := sf.Do(scoreboardVisibilityKey, func() (any, error) {
		if cached, ok := c.Get(scoreboardVisibilityKey); ok {
			return cached, nil
		}
		settings, err := getter.Get(context.WithoutCancel(ctx))
		if err != nil {
			return nil, err
		}
		c.Set(scoreboardVisibilityKey, settings.ScoreboardVisible)
		return settings.ScoreboardVisible, nil
	})
	if err != nil {
		return "", err
	}
	visibility, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("ScoreboardVisibility: unexpected type from singleflight: %T", v)
	}
	return visibility, nil
}
