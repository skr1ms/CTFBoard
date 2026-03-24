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

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"

	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
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

func (c *Controller) HandleWS(w http.ResponseWriter, r *http.Request) {
	user, ok := middleware.GetUser(r.Context())
	if !ok || user == nil {
		httputil.HandleError(w, r, httperr.ErrNotAuthenticated())
		return
	}

	opts := &websocket.AcceptOptions{
		OriginPatterns: c.allowedOrigins,
	}

	if len(c.allowedOrigins) == 0 {
		c.logger.Error("ws - HandleWS - ALLOWED_ORIGINS is not configured, rejecting connection")
		httputil.HandleError(w, r, httperr.ErrWebsocketOriginNotConfigured)
		return
	}
	if slices.Contains(c.allowedOrigins, "*") {
		c.logger.Error("ws - HandleWS - ALLOWED_ORIGINS=* is not allowed for security")
		httputil.HandleError(w, r, httperr.ErrWebsocketWildcardOriginNotAllowed)
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
