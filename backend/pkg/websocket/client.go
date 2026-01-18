package websocket

import (
	"context"
	"time"

	"github.com/coder/websocket"
)

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 512
)

func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.Unregister(c)
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.conn.SetReadLimit(maxMessageSize)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), pongWait)

		_, _, err := c.conn.Read(ctx)
		cancel()

		if err != nil {
			if websocket.CloseStatus(err) == websocket.StatusNormalClosure ||
				websocket.CloseStatus(err) == websocket.StatusGoingAway {
				return
			}
			return
		}
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case message, ok := <-c.send:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)

			if !ok {
				_ = c.conn.Close(websocket.StatusNormalClosure, "")
				cancel()
				return
			}

			w, err := c.conn.Writer(ctx, websocket.MessageText)
			if err != nil {
				cancel()
				return
			}

			_, _ = w.Write(message)

			n := len(c.send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				cancel()
				return
			}
			cancel()

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), writeWait)
			if err := c.conn.Ping(ctx); err != nil {
				cancel()
				return
			}
			cancel()
		}
	}
}
