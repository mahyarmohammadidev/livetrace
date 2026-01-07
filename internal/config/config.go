package config

import (
	"os"
	"strconv"
)

type Config struct {
	Addr string

	WriteWaitMS       int
	PongWaitMS        int
	PingPeriodMS      int
	MaxMessageBytes   int
	SendBuffer        int
	BroadcastBufSize  int
	RegisterBufSize   int
	UnregisterBufSize int

	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	RedisWorkers       int
	RedisQueueSize     int
	LocationTTLSeconds int
}

func Load() Config {
	return Config{
		Addr:              getEnv("ADDR", ":8080"),
		WriteWaitMS:       getEnvInt("WRITE_WAIT_MS", 5000),
		PongWaitMS:        getEnvInt("PONG_WAIT_MS", 60000),
		PingPeriodMS:      getEnvInt("PING_PERIOD_MS", 50000),
		MaxMessageBytes:   getEnvInt("MAX_MESSAGE_BYTES", 2048),
		SendBuffer:        getEnvInt("SEND_BUFFER", 128),
		BroadcastBufSize:  getEnvInt("BROADCAST_BUF", 4096),
		RegisterBufSize:   getEnvInt("REGISTER_BUF", 1024),
		UnregisterBufSize: getEnvInt("UNREGISTER_BUF", 1024),

		RedisAddr:          getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            getEnvInt("REDIS_DB", 0),
		RedisWorkers:       getEnvInt("REDIS_WORKERS", 8),
		RedisQueueSize:     getEnvInt("REDIS_QUEUE_SIZE", 100000),
		LocationTTLSeconds: getEnvInt("LOCATION_TTL_SECONDS", 120),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil {
			return n
		}
	}
	return fallback
}
