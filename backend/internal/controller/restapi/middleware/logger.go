package middleware

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/skr1ms/CTFBoard/pkg/logger"
)

func Logger(log logger.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			latency := time.Since(start)

			fields := map[string]any{
				"status":     ww.Status(),
				"method":     r.Method,
				"path":       r.URL.Path,
				"query":      r.URL.RawQuery,
				"ip":         r.RemoteAddr,
				"user_agent": r.UserAgent(),
				"latency_ms": latency.Milliseconds(),
				"bytes":      ww.BytesWritten(),
			}

			if reqID := middleware.GetReqID(r.Context()); reqID != "" {
				fields["request_id"] = reqID
			}

			reqLogger := log.WithFields(fields)

			if ww.Status() >= 500 {
				reqLogger.Error("http request failed")
			} else if ww.Status() >= 400 {
				reqLogger.Warn("http request error")
			} else {
				reqLogger.Info("http request")
			}
		})
	}
}
