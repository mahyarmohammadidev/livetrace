package redis

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"

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

const saveLocationLua = `
-- KEYS[1] = geoKey
-- KEYS[2] = userKey
-- KEYS[3] = lastSeenKey
-- ARGV[1] = userID
-- ARGV[2] = lng
-- ARGV[3] = lat
-- ARGV[4] = accuracy
-- ARGV[5] = ts
-- ARGV[6] = ttlMs

redis.call('GEOADD', KEYS[1], ARGV[2], ARGV[3], ARGV[1])

redis.call('HSET', KEYS[2],
  'lat', ARGV[3],
  'lng', ARGV[2],
  'accuracy', ARGV[4],
  'ts', ARGV[5]
)

local ttl = tonumber(ARGV[6])
if ttl and ttl > 0 then
  redis.call('PEXPIRE', KEYS[2], ttl)
else
  redis.call('PERSIST', KEYS[2])
end

redis.call('ZADD', KEYS[3], ARGV[5], ARGV[1])

return 1
`

func NewRedisStore(addr, password string, db int, ttl time.Duration) *RedisStore {
	poolSize := runtime.GOMAXPROCS(0) * 16
	if poolSize < 32 {
		poolSize = 32
	}
	if poolSize > 128 {
		poolSize = 128
	}

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

		saveScript: goredis.NewScript(saveLocationLua),
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
