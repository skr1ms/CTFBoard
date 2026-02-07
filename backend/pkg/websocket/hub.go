package websocket

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type broadcastItem struct {
	data []byte
	done chan struct{}
}

type Hub struct {
	clients      map[*Client]bool
	broadcast    chan broadcastItem
	register     chan *Client
	unregister   chan *Client
	clientCount  int64
	redisClient  *redis.Client
	redisChannel string
}

func NewHub(
	redisClient *redis.Client,
	redisChannel string,
) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan broadcastItem, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		redisClient:  redisClient,
		redisChannel: redisChannel,
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.clients[client] = true
			atomic.AddInt64(&h.clientCount, 1)
			if welcome, err := json.Marshal(Event{Type: "connected", Payload: nil, Timestamp: time.Now()}); err == nil {
				select {
				case client.send <- welcome:
				default:
				}
			}

		case client := <-h.unregister:
			h.unregisterClient(client)

		case item := <-h.broadcast:
			h.broadcastToClients(item)
		}
	}
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
		atomic.AddInt64(&h.clientCount, -1)
	}
}

func (h *Hub) broadcastToClients(item broadcastItem) {
	for client := range h.clients {
		select {
		case client.send <- item.data:
		default:
		}
	}
	if item.done != nil {
		close(item.done)
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- broadcastItem{data: data}
}

func (h *Hub) BroadcastEvent(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	done := make(chan struct{})
	select {
	case h.broadcast <- broadcastItem{data: data, done: done}:
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	case <-time.After(100 * time.Millisecond):
		return
	}
	if h.redisClient != nil {
		h.redisClient.Publish(context.Background(), h.redisChannel, data)
	}
}

func (h *Hub) SubscribeToRedis(ctx context.Context) {
	if h.redisClient == nil {
		return
	}
	pubsub := h.redisClient.Subscribe(ctx, h.redisChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-ch:
			if !ok {
				return
			}
			h.Broadcast([]byte(msg.Payload))
		}
	}
}

func (h *Hub) ClientCount() int {
	return int(atomic.LoadInt64(&h.clientCount))
}
