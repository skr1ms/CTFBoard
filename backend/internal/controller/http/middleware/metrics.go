package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	httpRequestsTotal   *prometheus.CounterVec
	httpRequestDuration *prometheus.HistogramVec
	metricsOnce         sync.Once
)

func initMetrics() {
	metricsOnce.Do(func() {
		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"method", "path", "status"},
		)

		httpRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		)

		prometheus.MustRegister(httpRequestsTotal, httpRequestDuration)
	})
}

func Metrics(next http.Handler) http.Handler {
	initMetrics()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		duration := time.Since(start).Seconds()
		status := strconv.Itoa(ww.Status())

		routeContext := chi.RouteContext(r.Context())
		path := ""
		if routeContext != nil && routeContext.RoutePattern() != "" {
			path = routeContext.RoutePattern()
		} else {
			if ww.Status() == http.StatusNotFound {
				path = "/not-found"
			} else {
				path = r.URL.Path
			}
		}

		method := r.Method

		httpRequestsTotal.WithLabelValues(method, path, status).Inc()
		httpRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	})
}
