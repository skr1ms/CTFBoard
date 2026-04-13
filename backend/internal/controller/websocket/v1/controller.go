package v1

import (
	"context"
	"net/http"
	"slices"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/wahrwelt-kit/go-httpkit/httputil"
	"github.com/wahrwelt-kit/go-logkit"
	"github.com/wahrwelt-kit/go-wskit"

	"github.com/TakuyaYagam1/AstroCTFb/internal/apperr"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/errmap"
	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
)

type Controller struct {
	hub            *wskit.Hub
	logger         logkit.Logger
	allowedOrigins []string
}

func NewController(
	hub *wskit.Hub,
	logger logkit.Logger,
	allowedOrigins []string,
) *Controller {
	return &Controller{
		hub:            hub,
		logger:         logger,
		allowedOrigins: allowedOrigins,
	}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Get("/ws", c.HandleWS)
}

// HandleWS upgrades the HTTP connection to a WebSocket and registers the client
// with the hub. Origin validation is applied before the upgrade: an empty
// allowedOrigins list or a wildcard (*) entry are both rejected as security
// misconfigurations. The connection runs on context.WithoutCancel so the hub
// can continue to deliver buffered messages after the originating request
// context is cancelled; read and write pumps are started in separate goroutines.
func (c *Controller) HandleWS(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrNotAuthenticated))

		return
	}

	opts := &websocket.AcceptOptions{
		OriginPatterns: c.allowedOrigins,
	}

	if len(c.allowedOrigins) == 0 {
		c.logger.Error("ws - HandleWS - ALLOWED_ORIGINS is not configured, rejecting connection")
		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrWebsocketOriginNotConfigured))

		return
	}

	if slices.Contains(c.allowedOrigins, "*") {
		c.logger.Error("ws - HandleWS - ALLOWED_ORIGINS=* is not allowed for security")
		httputil.HandleError(w, r, errmap.MapAppError(apperr.ErrWebsocketWildcardOriginNotAllowed))

		return
	}

	client, err := wskit.Accept(context.WithoutCancel(r.Context()), w, r, c.hub, opts)
	if err != nil {
		c.logger.WithError(err).Error("ws - HandleWS - Accept")

		return
	}

	go client.WritePump()
	go client.ReadPump()
}
