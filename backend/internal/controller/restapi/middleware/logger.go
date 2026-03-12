package middleware

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
)

const (
	statusServerError = http.StatusInternalServerError
	statusClientError = http.StatusBadRequest
)

var sensitiveQueryParams = map[string]struct{}{
	"token":         {},
	"state":         {},
	"code":          {},
	"password":      {},
	"secret":        {},
	"api_key":       {},
	"apikey":        {},
	"client_secret": {},
	"refresh_token": {},
	"access_token":  {},
	"authorization": {},
}

func redactQuery(raw string) string {
	if raw == "" {
		return ""
	}
	vals, err := url.ParseQuery(raw)
	if err != nil {
		return "[unparseable]"
	}
	redacted := false
	for key := range vals {
		if _, sensitive := sensitiveQueryParams[strings.ToLower(key)]; sensitive {
			vals[key] = []string{"[REDACTED]"}
			redacted = true
		}
	}
	if !redacted {
		return raw
	}
	return vals.Encode()
}

func Logger(log logger.Logger, trustedProxyCIDRs []string) func(next http.Handler) http.Handler {
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
				"ip":         httputil.GetClientIP(r, trustedProxyCIDRs),
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
