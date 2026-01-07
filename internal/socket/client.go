package socket

import (
	"encoding/json"
	"time"

	"github.com/gorilla/websocket"

	"livetrace/internal/config"
	"livetrace/internal/location"
	"livetrace/internal/redis"
)

type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	cfg    config.Config
	userID string

	writer *redis.RedisWriter
}

func NewClient(hub *Hub, conn *websocket.Conn, cfg config.Config, userID string, writer *redis.RedisWriter) *Client {
	return &Client{
		hub:    hub,
		conn:   conn,
		send:   make(chan []byte, cfg.SendBuffer),
		cfg:    cfg,
		userID: userID,
		writer: writer,
	}
}

func (c *Client) ReadPump() {
	defer func() {
		c.hub.unregister <- c
		_ = c.conn.Close()
	}()

	c.conn.SetReadLimit(int64(c.cfg.MaxMessageBytes))
	_ = c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.PongWaitMS) * time.Millisecond))
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(time.Duration(c.cfg.PongWaitMS) * time.Millisecond))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			return
		}
		c.hub.metrics.MessagesIn.Add(1)

		var msg LocationMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != MsgLocation {
			continue
		}

		msg.UserID = c.userID

		if !location.ValidLatLng(msg.Lat, msg.Lng) {
			continue
		}

		c.writer.Enqueue(redis.Location{
			UserID:   msg.UserID,
			Lat:      msg.Lat,
			Lng:      msg.Lng,
			Accuracy: msg.Accuracy,
			TS:       msg.TS,
		})

		c.hub.Broadcast(data)
	}
}

func (c *Client) WritePump() {
	ticker := time.NewTicker(time.Duration(c.cfg.PingPeriodMS) * time.Millisecond)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case msg, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.cfg.WriteWaitMS) * time.Millisecond))
			if !ok {
				_ = c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			_, _ = w.Write(msg)

			n := len(c.send)
			for range n {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

			c.hub.metrics.MessagesOut.Add(1)

		case <-ticker.C:
			_ = c.conn.SetWriteDeadline(time.Now().Add(time.Duration(c.cfg.WriteWaitMS) * time.Millisecond))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
