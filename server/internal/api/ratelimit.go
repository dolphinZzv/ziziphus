package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"ziziphus/pkg/model"
)

// LoginRateLimiter limits failed login attempts per IP+account.
// Uses Redis when rdb is set; falls back to in-memory for tests.
type LoginRateLimiter struct {
	mu           sync.Mutex
	attempts     map[string]*loginAttempt
	maxPerWindow int
	windowDur    time.Duration
	lockoutDur   time.Duration
	cleanupTick  time.Duration

	rdb redis.Cmdable
}

type loginAttempt struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

// NewLoginRateLimiter creates a login rate limiter.
// If rdb is nil, uses in-memory storage (suitable for tests / single-instance).
func NewLoginRateLimiter(maxAttempts int, window, lockout time.Duration, rdb redis.Cmdable) *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts:     make(map[string]*loginAttempt),
		maxPerWindow: maxAttempts,
		windowDur:    window,
		lockoutDur:   lockout,
		cleanupTick:  time.Minute,
		rdb:          rdb,
	}
	if rdb == nil {
		go rl.cleanupLoop()
	}
	return rl
}

// Allow checks whether a request from the given IP should be allowed.
// SetParams updates login rate-limiter parameters at runtime (hot-reload).
func (rl *LoginRateLimiter) SetParams(maxAttempts int, window, lockout time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.maxPerWindow = maxAttempts
	rl.windowDur = window
	rl.lockoutDur = lockout
}

func (rl *LoginRateLimiter) Allow(key string) error {
	if rl.rdb != nil {
		return rl.allowRedis(key)
	}
	return rl.allowMemory(key)
}

// Reset clears the rate limit for a given IP.
func (rl *LoginRateLimiter) Reset(ip string) {
	if rl.rdb != nil {
		rl.rdb.Del(context.Background(), "rl:login:"+ip, "rl:login:lock:"+ip)
		return
	}
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// allowRedis uses Redis INCR + EXPIRE for atomic window counting.
func (rl *LoginRateLimiter) allowRedis(key string) error {
	lockKey := "rl:login:lock:" + key
	winKey := "rl:login:" + key

	// Check lockout
	n, err := rl.rdb.Exists(context.Background(), lockKey).Result()
	if err == nil && n > 0 {
		return model.ErrRateLimited
	}

	// INCR — returns 1 on first creation
	count, err := rl.rdb.Incr(context.Background(), winKey).Result()
	if err != nil {
		return nil // fail open on Redis error
	}
	if count == 1 {
		rl.rdb.Expire(context.Background(), winKey, rl.windowDur)
	}

	if int(count) > rl.maxPerWindow {
		if rl.lockoutDur > 0 {
			rl.rdb.Set(context.Background(), lockKey, 1, rl.lockoutDur)
			rl.rdb.Del(context.Background(), winKey) // reset window so lockout expiry = fresh start
		}
		return model.ErrRateLimited
	}
	return nil
}

func (rl *LoginRateLimiter) allowMemory(key string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	a, ok := rl.attempts[key]
	if !ok {
		rl.attempts[key] = &loginAttempt{windowStart: now}
		a = rl.attempts[key]
	}

	// Reset window if expired
	if now.Sub(a.windowStart) > rl.windowDur {
		a.count = 0
		a.windowStart = now
	}

	// Check if currently locked out
	if now.Before(a.lockedUntil) {
		return model.ErrRateLimited
	}

	// If lockout has expired, reset the count so the user gets a fresh start
	if !a.lockedUntil.IsZero() {
		a.count = 0
		a.lockedUntil = time.Time{}
	}

	a.count++

	// Lock out if exceeded
	if a.count > rl.maxPerWindow {
		a.lockedUntil = now.Add(rl.lockoutDur)
		return model.ErrRateLimited
	}

	return nil
}

// Middleware creates an HTTP middleware that rate-limits requests by IP.
// Health, metrics and swagger paths are skipped (they are not login attempts).
func (rl *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health", "/metrics":
			next.ServeHTTP(w, r)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/swagger/") {
			next.ServeHTTP(w, r)
			return
		}
		ip := ClientIP(r)

		// Try to extract account from JSON body for rate limit key
		key := ip
		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if len(data) > 0 {
				var body map[string]any
				if json.Unmarshal(data, &body) == nil {
					if account, ok := body["account"].(string); ok && account != "" {
						key = ip + ":" + account
					}
				}
				r.Body = io.NopCloser(bytes.NewReader(data))
			} else {
				r.Body = io.NopCloser(bytes.NewReader(data))
			}
		}

		if err := rl.Allow(key); err != nil {
			Error(w, r, http.StatusTooManyRequests, err.(*model.AppError))
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (rl *LoginRateLimiter) Stop() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
}

func (rl *LoginRateLimiter) cleanupLoop() {
	for {
		time.Sleep(rl.cleanupTick)
		rl.mu.Lock()
		now := time.Now()
		for key, a := range rl.attempts {
			if now.Sub(a.windowStart) > rl.windowDur && now.After(a.lockedUntil) {
				delete(rl.attempts, key)
			}
		}
		rl.mu.Unlock()
	}
}

// ClientIP extracts the client IP from the TCP connection.
// X-Forwarded-For is NOT trusted for rate limiting to prevent spoofing.
func ClientIP(r *http.Request) string {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}
