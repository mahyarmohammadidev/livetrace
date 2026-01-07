package socket

import (
	"sync"

	"livetrace/internal/config"
)

type Hub struct {
	cfg     config.Config
	metrics *Metrics

	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte

	clients map[*Client]struct{}
	mu      sync.RWMutex
}

func NewHub(cfg config.Config, metrics *Metrics) *Hub {
	return &Hub{
		cfg:        cfg,
		metrics:    metrics,
		register:   make(chan *Client, cfg.RegisterBufSize),
		unregister: make(chan *Client, cfg.UnregisterBufSize),
		broadcast:  make(chan []byte, cfg.BroadcastBufSize),
		clients:    make(map[*Client]struct{}),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			h.clients[c] = struct{}{}
			h.mu.Unlock()

			h.metrics.ConnectedClients.Add(1)
			h.metrics.TotalConnections.Add(1)

		case c := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[c]; ok {
				delete(h.clients, c)
				close(c.send)
			}
			h.mu.Unlock()

			h.metrics.ConnectedClients.Add(-1)
			h.metrics.TotalDisconnects.Add(1)

		case msg := <-h.broadcast:
			h.metrics.Broadcasts.Add(1)

			h.mu.RLock()
			for c := range h.clients {
				select {
				case c.send <- msg:
					// ok
				default:
					// client too slow
					h.metrics.DroppedSendMessages.Add(1)
					go func(client *Client) { h.unregister <- client }(c)
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Broadcast(msg []byte) {
	select {
	case h.broadcast <- msg:
	default:
	}
}
