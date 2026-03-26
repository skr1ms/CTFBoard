package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sourcegraph/conc/pool"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
)

var trackingDroppedTotal = promauto.NewCounter(prometheus.CounterOpts{
	Name: "tracking_events_dropped_total",
	Help: "Total number of IP tracking events dropped because the channel was full.",
})

const (
	trackingDebounce      = 2 * time.Minute
	trackingWorkers       = 5
	trackingBufSize       = 256
	trackingCleanupPeriod = 10 * time.Minute
	trackingCtxTimeout    = 5 * time.Second
	trackingUserAgentMax  = 512
)

func sanitizeUserAgent(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if b.Len() >= trackingUserAgentMax {
			break
		}

		if r == unicode.ReplacementChar || r < 32 || r == 127 {
			continue
		}

		b.WriteRune(r)
	}

	return b.String()
}

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

func runTrackingJob(trackingUC usecase.TrackingUseCase, job trackingJob, log logkit.Logger) {
	tCtx, cancel := context.WithTimeout(context.Background(), trackingCtxTimeout)
	defer cancel()

	err := trackingUC.Track(tCtx, job.userID, job.ip, job.userAgent)
	if err != nil {
		log.WithError(err).Warn("middleware - IPTracking - Track: failed to track user")
	}
}

func IPTracking(ctx context.Context, trackingUC usecase.TrackingUseCase, log logkit.Logger) func(http.Handler) http.Handler {
	debouncer := &trackingDebouncer{lastSeen: make(map[uuid.UUID]time.Time)}
	ch := make(chan trackingJob, trackingBufSize)
	p := pool.New().WithMaxGoroutines(trackingWorkers)

	var trackingStopped atomic.Bool

	go func() {
		for job := range ch {
			p.Go(func() {
				runTrackingJob(trackingUC, job, log)
			})
		}

		p.Wait()
	}()

	go func() {
		<-ctx.Done()
		trackingStopped.Store(true)
		close(ch)
	}()

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
					ip:        kitMiddleware.GetClientIPFromContext(r.Context()),
					userAgent: sanitizeUserAgent(r.Header.Get("User-Agent")),
				}

				if !trackingStopped.Load() {
					func() {
						defer func() {
							if v := recover(); v != nil {
								log.WithFields(logkit.Fields{"panic": v}).Warn("middleware - IPTracking: recovered panic when sending tracking job")
							}
						}()

						select {
						case ch <- job:
						default:
							trackingDroppedTotal.Inc()
						}
					}()
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}
