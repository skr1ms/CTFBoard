package websocket

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

const (
	writeWait      = 10 * time.Second
	pingInterval   = 30 * time.Second
	maxMessageSize = 512
)

func NewClient(hub *Hub, conn *websocket.Conn, ctx context.Context) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		ctx:  ctx,
	}
}

func (c *Client) closeConn() {
	c.closeOnce.Do(func() {
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	})
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		c.closeConn()
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		_, _, err := c.conn.Read(c.ctx)
		if err != nil {
			return
		}
	}
}

//nolint:gocognit
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingInterval)
	defer func() {
		ticker.Stop()
		c.closeConn()
	}()

	for {
		select {
		case message, ok := <-c.send:
			ctx, cancel := context.WithTimeout(c.ctx, writeWait)

			if !ok {
				cancel()
				c.closeConn()
				return
			}

			w, err := c.conn.Writer(ctx, websocket.MessageText)
			if err != nil {
				cancel()
				return
			}

			if _, err := w.Write(message); err != nil {
				cancel()
				return
			}

			if err := w.Close(); err != nil {
				cancel()
				return
			}
			cancel()

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(c.ctx, writeWait)
			if err := c.conn.Ping(ctx); err != nil {
				cancel()
				return
			}
			cancel()

		case <-c.ctx.Done():
			return
		}
	}
}
