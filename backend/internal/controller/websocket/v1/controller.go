package ws

import (
	"net/http"
	"slices"

	"github.com/coder/websocket"
	"github.com/go-chi/chi/v5"
	"github.com/skr1ms/CTFBoard/pkg/logger"
	pkgWS "github.com/skr1ms/CTFBoard/pkg/websocket"
)

type Controller struct {
	hub            *pkgWS.Hub
	logger         logger.Interface
	allowedOrigins []string
}

func NewController(hub *pkgWS.Hub, logger logger.Interface, allowedOrigins []string) *Controller {
	return &Controller{
		hub:            hub,
		logger:         logger,
		allowedOrigins: allowedOrigins,
	}
}

func (c *Controller) RegisterRoutes(router chi.Router) {
	router.Get("/ws", c.HandleWS)
}

// @Summary      WebSocket connection
// @Description  Establishes WebSocket connection for real-time scoreboard updates
// @Tags         Events
// @Success      101  "Switching Protocols"
// @Router       /ws [get]
func (c *Controller) HandleWS(w http.ResponseWriter, r *http.Request) {
	opts := &websocket.AcceptOptions{
		OriginPatterns: c.allowedOrigins,
	}

	if len(c.allowedOrigins) == 0 || slices.Contains(c.allowedOrigins, "*") {
		opts.InsecureSkipVerify = true
	}

	conn, err := websocket.Accept(w, r, opts)
	if err != nil {
		c.logger.Error("ws - HandleWS - Accept", err)
		return
	}

	client := pkgWS.NewClient(c.hub, conn)
	c.hub.Register(client)

	go client.WritePump()
	go client.ReadPump()
}
