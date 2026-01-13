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
