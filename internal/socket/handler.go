package socket

import (
	"net/http"

	"github.com/gorilla/websocket"

	"livetrace/internal/config"
	"livetrace/internal/redis"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // TODO: restrict origin in prod
	},
}

func WSHandler(hub *Hub, cfg config.Config, writer *redis.RedisWriter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("userId")
		if userID == "" {
			http.Error(w, "missing userId", http.StatusBadRequest)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		client := NewClient(hub, conn, cfg, userID, writer)
		hub.register <- client

		go client.WritePump()
		go client.ReadPump()
	}
}
