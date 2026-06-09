package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	kitMiddleware "github.com/wahrwelt-kit/go-httpkit/httputil/middleware"
	"github.com/wahrwelt-kit/go-logkit"
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

// sanitizeUserAgent strips control characters (code points < 32 and DEL 127)
// and Unicode replacement characters from the User-Agent header value, then
// truncates the result to trackingUserAgentMax bytes to prevent excessively
// large strings from being stored in the tracking database.
func sanitizeUserAgent(s string) string {
	var b strings.Builder
	b.Grow(len(s))

	for _, r := range s {
		if r == unicode.ReplacementChar || r < 32 || r == 127 {
			continue
		}

		if b.Len()+utf8.RuneLen(r) > trackingUserAgentMax {
			break
		}

		b.WriteRune(r)
	}

	return b.String()
}

type trackingDebouncer struct {
	mu       sync.Mutex
	lastSeen map[uuid.UUID]time.Time
}

// shouldTrack returns true when userID has not been seen within trackingDebounce.
// Holds the mutex for the duration of both the check and the map update so
// concurrent callers for the same user never both pass the debounce window.
func (d *trackingDebouncer) shouldTrack(userID uuid.UUID) bool {
	d.mu.Lock()
	defer d.mu.Unlock()

	if t, ok := d.lastSeen[userID]; ok && time.Since(t) < trackingDebounce {
		return false
	}

	d.lastSeen[userID] = time.Now()

	return true
}

// purgeStale removes entries from the lastSeen map that are older than
// trackingDebounce. Called periodically to bound map growth for long-running
// deployments with large numbers of distinct users.
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

// ActivityTracker is the minimal interface required by IP tracking middleware.
type ActivityTracker interface {
	Track(ctx context.Context, userID uuid.UUID, ip, userAgent string) error
}

func runTrackingJob(ctx context.Context, trackingUC ActivityTracker, job trackingJob, log logkit.Logger) {
	tCtx, cancel := context.WithTimeout(ctx, trackingCtxTimeout)
	defer cancel()

	err := trackingUC.Track(tCtx, job.userID, job.ip, job.userAgent)
	if err != nil {
		log.WithError(err).Warn("middleware - IPTracking - Track: failed to track user")
	}
}

// IPTracking returns a middleware that asynchronously records authenticated user activity (IP, user-agent).
// Tracking is debounced per user (2 min) and dispatched to a bounded worker pool (5 goroutines) via a
// buffered channel (256). Dropped events increment a prometheus counter. The workers shut down cleanly
// when ctx is cancelled.
func IPTracking(ctx context.Context, trackingUC ActivityTracker, log logkit.Logger) func(http.Handler) http.Handler {
	debouncer := &trackingDebouncer{lastSeen: make(map[uuid.UUID]time.Time)}
	ch := make(chan trackingJob, trackingBufSize)

	var (
		trackingStopped atomic.Bool
		workerWG        sync.WaitGroup
	)

	workerWG.Add(trackingWorkers)

	for range trackingWorkers {
		go func() {
			defer workerWG.Done()

			for job := range ch {
				runTrackingJob(ctx, trackingUC, job, log)
			}
		}()
	}

	go func() {
		<-ctx.Done()
		trackingStopped.Store(true)
		close(ch)
		workerWG.Wait()
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
