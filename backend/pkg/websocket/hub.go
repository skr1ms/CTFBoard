package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/redis/go-redis/v9"
)

type Client struct {
	hub       *Hub
	conn      *websocket.Conn
	send      chan []byte
	ctx       context.Context
	closeOnce sync.Once
}

type broadcastItem struct {
	data []byte
}

type Hub struct {
	clients      map[*Client]bool
	broadcast    chan broadcastItem
	register     chan *Client
	unregister   chan *Client
	done         chan struct{}
	doneOnce     sync.Once
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
		register:     make(chan *Client, 64),
		unregister:   make(chan *Client, 64),
		done:         make(chan struct{}),
		redisClient:  redisClient,
		redisChannel: redisChannel,
	}
}

func (h *Hub) closeDone() {
	h.doneOnce.Do(func() { close(h.done) })
}

func (h *Hub) Run(ctx context.Context) {
	defer h.closeDone()
	for {
		select {
		case <-ctx.Done():
			for client := range h.clients {
				delete(h.clients, client)
				close(client.send)
				atomic.AddInt64(&h.clientCount, -1)
			}
			return
		case client := <-h.register:
			h.clients[client] = true
			atomic.AddInt64(&h.clientCount, 1)
			if welcome, err := json.Marshal(Event{Type: EventTypeConnected, Payload: nil, Timestamp: time.Now()}); err == nil {
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
}

func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.done:
	}
}

func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.done:
	}
}

func (h *Hub) Broadcast(data []byte) {
	t := time.NewTimer(5 * time.Second)
	defer t.Stop()
	select {
	case h.broadcast <- broadcastItem{data: data}:
	case <-h.done:
	case <-t.C:
	}
}

func (h *Hub) BroadcastEvent(ctx context.Context, event any) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	if h.redisClient != nil {
		pubCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := h.redisClient.Publish(pubCtx, h.redisChannel, data).Err()
		cancel()
		if err == nil {
			return
		}
	}
	h.Broadcast(data)
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
