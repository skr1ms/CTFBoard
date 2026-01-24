package websocket

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/coder/websocket"
	"github.com/skr1ms/CTFBoard/pkg/redis"
)

type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte
}

type Hub struct {
	clients      map[*Client]bool
	broadcast    chan []byte
	register     chan *Client
	unregister   chan *Client
	clientCount  int64
	redisClient  redis.Client
	redisChannel string
}

func NewHub(redisClient redis.Client, redisChannel string) *Hub {
	return &Hub{
		clients:      make(map[*Client]bool),
		broadcast:    make(chan []byte, 256),
		register:     make(chan *Client),
		unregister:   make(chan *Client),
		redisClient:  redisClient,
		redisChannel: redisChannel,
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			atomic.AddInt64(&h.clientCount, 1)

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
				atomic.AddInt64(&h.clientCount, -1)
			}

		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
					atomic.AddInt64(&h.clientCount, -1)
				}
			}
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- data
}

func (h *Hub) BroadcastEvent(event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if h.redisClient != nil {
		h.redisClient.Publish(context.Background(), h.redisChannel, data)
	} else {
		h.Broadcast(data)
	}
}

func (h *Hub) SubscribeToRedis(ctx context.Context) {
	if h.redisClient == nil {
		return
	}
	pubsub := h.redisClient.Subscribe(ctx, h.redisChannel)
	defer func() { _ = pubsub.Close() }()

	ch := pubsub.Channel()
	for msg := range ch {
		h.Broadcast([]byte(msg.Payload))
	}
}

func (h *Hub) ClientCount() int {
	return int(atomic.LoadInt64(&h.clientCount))
}
