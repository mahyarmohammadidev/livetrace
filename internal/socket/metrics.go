package socket

import (
	"encoding/json"
	"net/http"
	"sync/atomic"
	"time"
)

type Metrics struct {
	ConnectedClients atomic.Int64

	TotalConnections    atomic.Uint64
	TotalDisconnects    atomic.Uint64
	MessagesIn          atomic.Uint64
	MessagesOut         atomic.Uint64
	Broadcasts          atomic.Uint64
	DroppedSlowClients  atomic.Uint64
	DroppedSendMessages atomic.Uint64

	RedisEnqueued atomic.Uint64
	RedisDropped  atomic.Uint64
	RedisWritten  atomic.Uint64
	RedisErrors   atomic.Uint64

	StartTime time.Time
}

func NewMetrics() *Metrics {
	return &Metrics{StartTime: time.Now()}
}

type metricsSnapshot struct {
	UptimeSeconds float64 `json:"uptimeSeconds"`

	ConnectedClients int64 `json:"connectedClients"`

	TotalConnections uint64 `json:"totalConnections"`
	TotalDisconnects uint64 `json:"totalDisconnects"`

	MessagesIn  uint64 `json:"messagesIn"`
	MessagesOut uint64 `json:"messagesOut"`

	Broadcasts          uint64 `json:"broadcasts"`
	DroppedSlowClients  uint64 `json:"droppedSlowClients"`
	DroppedSendMessages uint64 `json:"droppedSendMessages"`

	RedisEnqueued uint64 `json:"redisEnqueued"`
	RedisDropped  uint64 `json:"redisDropped"`
	RedisWritten  uint64 `json:"redisWritten"`
	RedisErrors   uint64 `json:"redisErrors"`
}

func (m *Metrics) Snapshot() metricsSnapshot {
	uptime := time.Since(m.StartTime).Seconds()
	return metricsSnapshot{
		UptimeSeconds: uptime,

		ConnectedClients: m.ConnectedClients.Load(),

		TotalConnections: m.TotalConnections.Load(),
		TotalDisconnects: m.TotalDisconnects.Load(),

		MessagesIn:  m.MessagesIn.Load(),
		MessagesOut: m.MessagesOut.Load(),

		Broadcasts:          m.Broadcasts.Load(),
		DroppedSlowClients:  m.DroppedSlowClients.Load(),
		DroppedSendMessages: m.DroppedSendMessages.Load(),

		RedisEnqueued: m.RedisEnqueued.Load(),
		RedisDropped:  m.RedisDropped.Load(),
		RedisWritten:  m.RedisWritten.Load(),
		RedisErrors:   m.RedisErrors.Load(),
	}
}

func MetricsHandler(m *Metrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := m.Snapshot()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(snap)
	}
}
