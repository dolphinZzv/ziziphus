package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"ziziphus/pkg/model"
)

// RegisterLimiter limits new account registrations per IP.
// Stricter than LoginRateLimiter because an attacker can try
// many different account names from the same IP.
// Uses Redis when rdb is set; falls back to in-memory for tests.
type RegisterLimiter struct {
	mu          sync.Mutex
	entries     map[string]*regCounter
	maxPerWin   int
	windowDur   time.Duration
	cleanupTick time.Duration

	rdb redis.Cmdable
}

type regCounter struct {
	count       int
	windowStart time.Time
}

func NewRegisterLimiter(maxPerWindow int, window time.Duration, rdb redis.Cmdable) *RegisterLimiter {
	rl := &RegisterLimiter{
		entries:     make(map[string]*regCounter),
		maxPerWin:   maxPerWindow,
		windowDur:   window,
		cleanupTick: 5 * time.Minute,
		rdb:         rdb,
	}
	if rdb == nil {
		go rl.cleanupLoop()
	}
	return rl
}

func (rl *RegisterLimiter) Allow(ip string) error {
	if rl.rdb != nil {
		return rl.allowRedis(ip)
	}
	return rl.allowMemory(ip)
}

func (rl *RegisterLimiter) allowRedis(ip string) error {
	key := "rl:reg:" + ip
	count, err := rl.rdb.Incr(context.Background(), key).Result()
	if err != nil {
		return nil // fail open on Redis error
	}
	if count == 1 {
		rl.rdb.Expire(context.Background(), key, rl.windowDur)
	}
	if int(count) > rl.maxPerWin {
		return model.ErrRateLimited
	}
	return nil
}

func (rl *RegisterLimiter) allowMemory(ip string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	c, ok := rl.entries[ip]
	if !ok {
		rl.entries[ip] = &regCounter{windowStart: now}
		c = rl.entries[ip]
	}

	if now.Sub(c.windowStart) > rl.windowDur {
		c.count = 0
		c.windowStart = now
	}

	c.count++
	if c.count > rl.maxPerWin {
		return model.ErrRateLimited
	}
	return nil
}

func (rl *RegisterLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)
		if err := rl.Allow(ip); err != nil {
			Error(w, r, http.StatusTooManyRequests, model.ErrRateLimited)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (rl *RegisterLimiter) cleanupLoop() {
	for {
		time.Sleep(rl.cleanupTick)
		rl.mu.Lock()
		now := time.Now()
		for ip, c := range rl.entries {
			if now.Sub(c.windowStart) > rl.windowDur {
				delete(rl.entries, ip)
			}
		}
		rl.mu.Unlock()
	}
}
