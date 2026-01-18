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
	restapiRequestsTotal   *prometheus.CounterVec
	restapiRequestDuration *prometheus.HistogramVec
	metricsOnce            sync.Once
)

func initMetrics() {
	metricsOnce.Do(func() {
		restapiRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "restapi_requests_total",
				Help: "Total number of restapi requests",
			},
			[]string{"method", "path", "status"},
		)

		restapiRequestDuration = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "restapi_request_duration_seconds",
				Help:    "restapi request duration in seconds",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"method", "path", "status"},
		)

		prometheus.MustRegister(restapiRequestsTotal, restapiRequestDuration)
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

		restapiRequestsTotal.WithLabelValues(method, path, status).Inc()
		restapiRequestDuration.WithLabelValues(method, path, status).Observe(duration)
	})
}
