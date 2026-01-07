package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"math/rand"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
)

type LocationMessage struct {
	Type     string  `json:"type"`
	UserID   string  `json:"userId"`
	Lat      float64 `json:"lat"`
	Lng      float64 `json:"lng"`
	Accuracy float64 `json:"accuracy"`
	TS       int64   `json:"ts"`
}

func main() {
	var (
		url       = flag.String("url", "ws://localhost:8080/ws", "WebSocket URL (without userId)")
		clients   = flag.Int("clients", 200, "Number of concurrent clients")
		interval  = flag.Int("interval", 1000, "Send interval in ms per client")
		centerLat = flag.Float64("lat", 35.6892, "Center latitude (Tehran default)")
		centerLng = flag.Float64("lng", 51.3890, "Center longitude (Tehran default)")
		spread    = flag.Float64("spread", 0.02, "Spread in degrees (~0.02 is a few km)")
	)
	flag.Parse()

	rand.Seed(time.Now().UnixNano())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// handle ctrl+c
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	log.Printf("Starting loadgen: clients=%d interval=%dms", *clients, *interval)
	log.Printf("Center: lat=%.6f lng=%.6f spread=%.4f", *centerLat, *centerLng, *spread)

	var wg sync.WaitGroup
	wg.Add(*clients)

	for i := 0; i < *clients; i++ {
		id := fmt.Sprintf("iran-%d", i)

		// random start positions in Tehran-ish
		lat := *centerLat + (rand.Float64()*2-1)*(*spread)
		lng := *centerLng + (rand.Float64()*2-1)*(*spread)

		go func(userID string, startLat, startLng float64) {
			defer wg.Done()
			runClient(ctx, *url, userID, startLat, startLng, time.Duration(*interval)*time.Millisecond)
		}(id, lat, lng)
	}

	<-stop
	log.Println("Stopping loadgen...")
	cancel()
	wg.Wait()
	log.Println("All clients stopped.")
}

func runClient(ctx context.Context, baseURL, userID string, lat, lng float64, interval time.Duration) {
	// Each client gets its own WS connection
	url := fmt.Sprintf("%s?userId=%s", baseURL, userID)

	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
	}

	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		log.Printf("[%s] dial error: %v", userID, err)
		return
	}
	defer conn.Close()

	// writer loop
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Simulated movement params
	step := 0.0002 + rand.Float64()*0.0003 // small movement per tick
	angle := rand.Float64() * 2 * math.Pi

	for {
		select {
		case <-ctx.Done():
			_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "bye"))
			return

		case <-ticker.C:
			// Move in a slow “walk” direction, slightly wobble angle
			angle += (rand.Float64()*2 - 1) * 0.05
			lat += math.Sin(angle) * step
			lng += math.Cos(angle) * step

			msg := LocationMessage{
				Type:     "location",
				UserID:   userID,
				Lat:      lat,
				Lng:      lng,
				Accuracy: 5 + rand.Float64()*5,
				TS:       time.Now().Unix(),
			}

			data, _ := json.Marshal(msg)
			if err := conn.WriteMessage(websocket.TextMessage, data); err != nil {
				// try reconnect (simple retry)
				log.Printf("[%s] write error: %v (reconnecting)", userID, err)
				reconnect(ctx, &conn, url)
			}
		}
	}
}

func reconnect(ctx context.Context, conn **websocket.Conn, url string) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		dialer := websocket.Dialer{HandshakeTimeout: 5 * time.Second}
		c, _, err := dialer.Dial(url, nil)
		if err == nil {
			*conn = c
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
}
