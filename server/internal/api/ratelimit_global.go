package api

import (
	"context"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"ziziphus/pkg/model"
)

// GlobalRateLimiter is a per-IP token bucket applied to all routes
// as a basic DDoS / brute-force deterrent.
// Uses a Redis Lua script for atomic token bucket when rdb is set.
type GlobalRateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*gbBucket
	rate    int
	burst   int

	rdb     redis.Cmdable
	luaHash string // SHA1 of the token bucket Lua script
}

type gbBucket struct {
	tokens    float64
	lastCheck time.Time
}

// tokenBucketLua atomically refills and consumes one token from a Redis token bucket.
// KEYS[1] = token counter key, KEYS[2] = last-refill timestamp key
// ARGV[1] = rate (tokens/sec), ARGV[2] = burst, ARGV[3] = now (ms)
// Returns 1 if allowed, 0 if rate limited.
const tokenBucketLua = `
local tk = redis.call("GET", KEYS[1])
local ts = redis.call("GET", KEYS[2])
if not tk then
	tk = tonumber(ARGV[2])
	ts = tonumber(ARGV[3])
else
	tk = tonumber(tk)
	ts = tonumber(ts)
	local elapsed = (tonumber(ARGV[3]) - ts) / 1000.0
	tk = tk + elapsed * tonumber(ARGV[1])
	if tk > tonumber(ARGV[2]) then tk = tonumber(ARGV[2]) end
end
if tk < 1 then return 0 end
tk = tk - 1
redis.call("SET", KEYS[1], tk)
redis.call("SET", KEYS[2], ARGV[3])
return 1
`

func NewGlobalRateLimiter(rate, burst int, rdb redis.Cmdable) *GlobalRateLimiter {
	rl := &GlobalRateLimiter{
		buckets: make(map[string]*gbBucket),
		rate:    rate,
		burst:   burst,
		rdb:     rdb,
	}
	if rdb != nil {
		// Load Lua script once
		hash, err := rdb.ScriptLoad(context.Background(), tokenBucketLua).Result()
		if err == nil {
			rl.luaHash = hash
		}
	}
	return rl
}

func (rl *GlobalRateLimiter) Allow(ip string) error {
	if rl.rdb != nil {
		return rl.allowRedis(ip)
	}
	return rl.allowMemory(ip)
}

func (rl *GlobalRateLimiter) allowRedis(ip string) error {
	key := "rl:global:" + ip
	tsKey := key + ":ts"

	var n int64
	var err error

	if rl.luaHash != "" {
		n, err = rl.rdb.EvalSha(context.Background(), rl.luaHash, []string{key, tsKey},
			rl.rate, rl.burst, time.Now().UnixMilli()).Int64()
	} else {
		n, err = rl.rdb.Eval(context.Background(), tokenBucketLua, []string{key, tsKey},
			rl.rate, rl.burst, time.Now().UnixMilli()).Int64()
	}
	if err != nil {
		return nil // fail open on Redis error
	}
	if n == 0 {
		return model.ErrRateLimited
	}
	return nil
}

func (rl *GlobalRateLimiter) allowMemory(ip string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	b, ok := rl.buckets[ip]
	if !ok {
		b = &gbBucket{tokens: float64(rl.burst), lastCheck: time.Now()}
		rl.buckets[ip] = b
	}

	now := time.Now()
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * float64(rl.rate)
	if b.tokens > float64(rl.burst) {
		b.tokens = float64(rl.burst)
	}
	b.lastCheck = now

	if b.tokens < 1 {
		return model.ErrRateLimited
	}
	b.tokens--
	return nil
}

// Middleware returns an HTTP middleware. Health, metrics and swagger paths are skipped.
func (rl *GlobalRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || path == "/metrics" || strings.HasPrefix(path, "/swagger/") {
			next.ServeHTTP(w, r)
			return
		}
		ip := ClientIP(r)
		if err := rl.Allow(ip); err != nil {
			w.Header().Set("Retry-After", "1")
			http.Error(w, "rate limited", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}
