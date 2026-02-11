package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/skr1ms/CTFBoard/internal/entity"
	"github.com/skr1ms/CTFBoard/internal/repo"
	"github.com/skr1ms/CTFBoard/pkg/httputil"
)

type scoreboardVisibilityCache struct {
	mu         sync.RWMutex
	visibility string
	fetchedAt  time.Time
	ttl        time.Duration
}

func (c *scoreboardVisibilityCache) get(ctx context.Context, repo repo.AppSettingsRepository) (string, error) {
	c.mu.RLock()
	if time.Since(c.fetchedAt) < c.ttl && c.visibility != "" {
		visibility := c.visibility
		c.mu.RUnlock()
		return visibility, nil
	}
	c.mu.RUnlock()

	settings, err := repo.Get(ctx)
	if err != nil {
		return "", err
	}

	c.mu.Lock()
	c.visibility = settings.ScoreboardVisible
	c.fetchedAt = time.Now()
	c.mu.Unlock()

	return settings.ScoreboardVisible, nil
}

func ScoreboardVisibility(appSettingsRepo repo.AppSettingsRepository) func(http.Handler) http.Handler {
	cache := &scoreboardVisibilityCache{
		ttl: 30 * time.Second,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, hasUser := GetUser(r.Context())
			if hasUser && user.Role == entity.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			visibility, err := cache.get(r.Context(), appSettingsRepo)
			if err != nil {
				httputil.RenderError(w, r, http.StatusInternalServerError, "failed to get scoreboard visibility settings")
				return
			}

			switch visibility {
			case entity.ScoreboardVisiblePublic:
				next.ServeHTTP(w, r)
			case entity.ScoreboardVisibleHidden:
				httputil.RenderError(w, r, http.StatusForbidden, "scoreboard is currently hidden")
			case entity.ScoreboardVisibleAdminsOnly:
				httputil.RenderError(w, r, http.StatusForbidden, "scoreboard is only available to administrators")
			default:
				next.ServeHTTP(w, r)
			}
		})
	}
}
