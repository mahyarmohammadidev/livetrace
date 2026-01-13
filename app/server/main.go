package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"livetrace/internal/config"
	"livetrace/internal/middleware"
	"livetrace/internal/redis"
	"livetrace/internal/socket"
)

func main() {
	cfg := config.Load()
	locationTTL := time.Duration(cfg.LocationTTLSeconds) * time.Second
	store := redis.NewRedisStore(cfg.RedisAddr, cfg.RedisPassword, cfg.RedisDB, locationTTL)
	writer := redis.NewRedisWriter(store, redis.RedisWriterConfig{
		QueueSize: 100_000,
		Workers:   4,
	})
	store.StartCleanupLoop(10 * time.Second)
	metrics := socket.NewMetrics()
	hub := socket.NewHub(cfg, metrics)
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", socket.WSHandler(hub, cfg, writer))
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	mux.Handle("/", http.FileServer(http.Dir("./web")))
	mux.HandleFunc("/metrics", socket.MetricsHandler(metrics))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           middleware.Recovery(mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("WebSocket server listening on %s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	log.Println("Shutting down...")
	_ = server.Close()
}
