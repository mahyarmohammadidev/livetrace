package redis

import (
	"context"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

func (s *RedisStore) StartCleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)

	go func() {
		for range ticker.C {
			s.cleanupStaleUsers()
		}
	}()
}

func (s *RedisStore) cleanupStaleUsers() {
	cutoff := time.Now().Unix() - int64(s.ttl.Seconds())
	ctx := context.Background()
	stale, err := s.rdb.ZRangeByScore(ctx, s.lastSeenKey, &redis.ZRangeBy{
		Min: "-inf",
		Max: strconv.FormatInt(cutoff, 10),
	}).Result()

	if err != nil || len(stale) == 0 {
		return
	}

	pipe := s.rdb.Pipeline()

	pipe.ZRem(ctx, s.geoKey, stale)
	pipe.ZRem(ctx, s.lastSeenKey, stale)

	_, _ = pipe.Exec(ctx)
}
