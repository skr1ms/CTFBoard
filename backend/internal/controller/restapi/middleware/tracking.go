package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/TakuyaYagam1/AstroCTFb/internal/usecase"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

var (
	trackingDroppedTotal prometheus.Counter
	trackingDroppedOnce  sync.Once
)

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

func initTrackingMetrics() {
	trackingDroppedOnce.Do(func() {
		trackingDroppedTotal = prometheus.NewCounter(prometheus.CounterOpts{
			Name: "tracking_events_dropped_total",
			Help: "Total number of IP tracking events dropped because the channel was full.",
		})
		prometheus.MustRegister(trackingDroppedTotal)
	})
}

func runTrackingJob(trackingUC usecase.TrackingUseCase, job trackingJob, log logger.Logger) {
	tCtx, cancel := context.WithTimeout(context.Background(), trackingCtxTimeout)
	defer cancel()
	if err := trackingUC.Track(tCtx, job.userID, job.ip, job.userAgent); err != nil {
		log.WithError(err).Warn("middleware - IPTracking - Track: failed to track user")
	}
}

func IPTracking(ctx context.Context, trackingUC usecase.TrackingUseCase, trustedProxyCIDRs []string, log logger.Logger) func(http.Handler) http.Handler {
	initTrackingMetrics()
	debouncer := &trackingDebouncer{lastSeen: make(map[uuid.UUID]time.Time)}
	ch := make(chan trackingJob, trackingBufSize)
	var wg sync.WaitGroup

	wg.Add(trackingWorkers)
	for range trackingWorkers {
		go func() {
			defer wg.Done()
			for job := range ch {
				runTrackingJob(trackingUC, job, log)
			}
		}()
	}

	go func() {
		<-ctx.Done()
		close(ch)
		wg.Wait()
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
					ip:        httputil.GetClientIP(r, trustedProxyCIDRs),
					userAgent: sanitizeUserAgent(r.Header.Get("User-Agent")),
				}
				func() {
					defer func() {
						if v := recover(); v != nil {
							log.WithFields(logger.Fields{"panic": v}).Warn("middleware - IPTracking: recovered panic when sending tracking job")
						}
					}()
					select {
					case ch <- job:
					default:
						trackingDroppedTotal.Inc()
					}
				}()
			}
			next.ServeHTTP(w, r)
		})
	}
}
