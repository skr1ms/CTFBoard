package sse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"
)

func TestNewSSEHandler(t *testing.T) {
	t.Parallel()

	hub := wskit.NewHub()
	logger := logkit.Noop()

	h := NewSSEHandler(hub, logger)

	assert.Same(t, hub, h.hub)
	assert.Equal(t, logger, h.logger)
}

func TestServeHTTPReturnsWhenRequestContextIsCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	req := httptest.NewRequestWithContext(ctx, http.MethodGet, "/events", http.NoBody)
	w := httptest.NewRecorder()
	h := NewSSEHandler(wskit.NewHub(), logkit.Noop())

	h.ServeHTTP(w, req)

	assert.Equal(t, "text/event-stream", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", w.Header().Get("Connection"))
	assert.Equal(t, "no", w.Header().Get("X-Accel-Buffering"))
}
