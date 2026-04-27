package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
)

// responseBuffer wraps http.ResponseWriter to capture the status code and body
// written by the next handler without flushing them to the client immediately.
// Header mutations still reach the original writer's header map directly.
type responseBuffer struct {
	http.ResponseWriter

	statusCode int
	body       bytes.Buffer
}

func (rb *responseBuffer) WriteHeader(code int) {
	rb.statusCode = code
}

func (rb *responseBuffer) Write(b []byte) (int, error) {
	return rb.body.Write(b)
}

// ETag adds conditional GET support to cacheable GET/HEAD endpoints.
//
// It buffers the response, computes a SHA-256 ETag, and short-circuits
// with 304 Not Modified when the client's If-None-Match matches.
// Only applied to 200 OK responses; error responses are forwarded as-is.
// POST/PUT/DELETE requests are passed through unchanged.
func ETag(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)

			return
		}

		buf := &responseBuffer{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(buf, r)

		if buf.statusCode != http.StatusOK {
			w.WriteHeader(buf.statusCode)

			if r.Method != http.MethodHead {
				_, _ = w.Write(buf.body.Bytes())
			}

			return
		}

		sum := sha256.Sum256(buf.body.Bytes())
		etag := `"` + hex.EncodeToString(sum[:]) + `"`
		w.Header().Set("ETag", etag)

		if r.Header.Get("If-None-Match") == etag {
			w.WriteHeader(http.StatusNotModified)

			return
		}

		w.WriteHeader(http.StatusOK)

		if r.Method != http.MethodHead {
			_, _ = w.Write(buf.body.Bytes())
		}
	})
}
