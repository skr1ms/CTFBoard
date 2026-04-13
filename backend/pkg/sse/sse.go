package sse

import (
	"net/http"

	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"
)

// SSEHandler is an http.Handler that upgrades HTTP connections to Server-Sent Events
// using go-wskit as an alternative transport for clients that cannot use WebSocket.
type SSEHandler struct {
	hub    *wskit.Hub
	logger logkit.Logger
}

// NewSSEHandler creates an SSEHandler backed by the given wskit.Hub.
func NewSSEHandler(hub *wskit.Hub, logger logkit.Logger) *SSEHandler {
	return &SSEHandler{hub: hub, logger: logger}
}

// ServeHTTP upgrades the request to an SSE stream via wskit.AcceptSSE.
func (h *SSEHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := wskit.AcceptSSE(w, r, h.hub); err != nil {
		h.logger.WithError(err).Warn("sse - ServeHTTP - AcceptSSE")
	}
}
