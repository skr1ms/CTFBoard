package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/google/uuid"
)

const (
	trackingDebounce      = 2 * time.Minute
	trackingWorkers       = 5
	trackingBufSize       = 256
	trackingCleanupPeriod = 10 * time.Minute
	trackingCtxTimeout    = 5 * time.Second
)

type trackingDebouncer struct {
	mu       sync.Mutex
	lastSeen map[uuid.UUID]time.Time
}

func (d *trackingDebouncer) shouldTrack(userID uuid.UUID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	if t, ok := d.lastSeen[userID]; ok && time.Since(t) < trackingDebounce {
		return false
	}
	d.lastSeen[userID] = time.Now()
	return true
}

func (d *trackingDebouncer) purgeStale() {
	d.mu.Lock()
	defer d.mu.Unlock()
	cutoff := time.Now().Add(-trackingDebounce)
	for id, t := range d.lastSeen {
		if t.Before(cutoff) {
			delete(d.lastSeen, id)
		}
	}
}

type trackingJob struct {
	userID    uuid.UUID
	ip        string
	userAgent string
}

func runTrackingJob(trackingUC usecase.TrackingUseCase, job trackingJob, log logger.Logger) {
	tCtx, cancel := context.WithTimeout(context.Background(), trackingCtxTimeout)
	defer cancel()
	if err := trackingUC.Track(tCtx, job.userID, job.ip, job.userAgent); err != nil {
		log.WithError(err).Warn("middleware - IPTracking - Track: failed to track user")
	}
}

func IPTracking(ctx context.Context, trackingUC usecase.TrackingUseCase, trustedProxyCIDRs []string, log logger.Logger) func(http.Handler) http.Handler {
	debouncer := &trackingDebouncer{lastSeen: make(map[uuid.UUID]time.Time)}
	ch := make(chan trackingJob, trackingBufSize)

	for range trackingWorkers {
		go func() {
			for {
				select {
				case job, ok := <-ch:
					if !ok {
						return
					}
					runTrackingJob(trackingUC, job, log)
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	go func() {
		ticker := time.NewTicker(trackingCleanupPeriod)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				debouncer.purgeStale()
			case <-ctx.Done():
				return
			}
		}
	}()

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, ok := GetUser(r.Context())
			if ok && user != nil && debouncer.shouldTrack(user.ID) {
				job := trackingJob{
					userID:    user.ID,
					ip:        httputil.GetClientIP(r, trustedProxyCIDRs),
					userAgent: r.Header.Get("User-Agent"),
				}
				select {
				case ch <- job:
				default:
					// channel full: drop tracking entry rather than blocking the request
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
