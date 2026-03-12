package v1

import (
	"net/http"
	"slices"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"

	"github.com/TakuyaYagam1/AstroCTFb/internal/controller/restapi/middleware"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httperr"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/httputil"
	"github.com/TakuyaYagam1/AstroCTFb/pkg/logger"
	pkgWS "github.com/TakuyaYagam1/AstroCTFb/pkg/websocket"
)

type Controller struct {
	hub            *pkgWS.Hub
	logger         logger.Logger
	allowedOrigins []string
}

func NewController(
	hub *pkgWS.Hub,
	logger logger.Logger,
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
		httputil.HandleError(w, r, httperr.ErrNotAuthenticated)
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

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		c.logger.WithError(err).Error("ws - HandleWS - Accept")
		return
	}

	client := pkgWS.NewClient(c.hub, conn, r.Context())
	c.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
