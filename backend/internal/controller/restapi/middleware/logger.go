package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	statusServerError = http.StatusInternalServerError
	statusClientError = http.StatusBadRequest
)

var sensitiveQueryParams = map[string]struct{}{
	"token": {},
	"state": {},
	"code":  {},
}

func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	hasSensitive := false
	for key := range sensitiveQueryParams {
		if strings.Contains(strings.ToLower(raw), key+"=") {
			hasSensitive = true
			break
		}
	}
	if !hasSensitive {
		return raw
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return "[unparseable]"
	}
	for key := range vals {
		if _, sensitive := sensitiveQueryParams[strings.ToLower(key)]; sensitive {
			vals[key] = []string{"[REDACTED]"}
		}
	}
	return vals.Encode()
}

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
				"query":      redactQuery(r.URL.RawQuery),
				"ip":         r.RemoteAddr,
				"user_agent": r.UserAgent(),
				"latency_ms": latency.Milliseconds(),
				"bytes":      ww.BytesWritten(),
			}

			if reqID := middleware.GetReqID(r.Context()); reqID != "" {
				fields["request_id"] = reqID
			}

			reqLogger := log.WithFields(fields)

			if ww.Status() >= statusServerError {
				reqLogger.Error("http request failed")
			} else if ww.Status() >= statusClientError {
				reqLogger.Warn("http request error")
			} else {
				reqLogger.Info("http request")
			}
		})
	}
}
