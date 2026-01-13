package redis

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

	_ "embed"

	goredis "github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *goredis.Client

	geoKey      string
	lastSeenKey string
	ttl         time.Duration

	opTimeout time.Duration
	inflight  chan struct{}

	saveScript *goredis.Script
}

type Location struct {
	UserID   string
	Lat      float64
	Lng      float64
	Accuracy float64
	TS       int64
}

//go:embed storage.lua
var storeScript string

func NewRedisStore(addr, password string, db int, ttl time.Duration) *RedisStore {
	poolSize := min(max(runtime.GOMAXPROCS(0)*16, 32), 128)

	rdb := goredis.NewClient(&goredis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,

		PoolSize:     poolSize,
		MinIdleConns: poolSize / 4,

		PoolTimeout: 1 * time.Second,

		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		MaxRetries:      1,
		MinRetryBackoff: 25 * time.Millisecond,
		MaxRetryBackoff: 250 * time.Millisecond,
	})

	return &RedisStore{
		rdb:         rdb,
		geoKey:      "geo:users",
		lastSeenKey: "users:last_seen",
		ttl:         ttl,

		opTimeout: 5 * time.Second,

		inflight: make(chan struct{}, poolSize),

		saveScript: goredis.NewScript(storeScript),
	}
}

func (s *RedisStore) Close() error {
	return s.rdb.Close()
}

func (s *RedisStore) SaveLocation(loc Location) error {
	return s.SaveLocationCtx(context.Background(), loc)
}

func (s *RedisStore) SaveLocationCtx(ctx context.Context, loc Location) error {
	if ctx == nil {
		ctx = context.Background()
	}

	if _, hasDeadline := ctx.Deadline(); !hasDeadline && s.opTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, s.opTimeout)
		defer cancel()
	}

	select {
	case s.inflight <- struct{}{}:
		defer func() { <-s.inflight }()
	case <-ctx.Done():
		return ctx.Err()
	}

	userKey := fmt.Sprintf("loc:user:%s", loc.UserID)
	ttlMs := s.ttl.Milliseconds()

	keys := []string{s.geoKey, userKey, s.lastSeenKey}
	err := s.saveScript.Run(
		ctx,
		s.rdb,
		keys,
		loc.UserID,
		loc.Lng,
		loc.Lat,
		loc.Accuracy,
		loc.TS,
		ttlMs,
	).Err()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return fmt.Errorf("redis SaveLocation timeout/cancel: %w", err)
		}
		return fmt.Errorf("redis SaveLocation failed: %w", err)
	}

	return nil
}
