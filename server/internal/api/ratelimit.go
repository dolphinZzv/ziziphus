package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"siciv.space/agent/panda_ai/pkg/model"
)

// LoginRateLimiter limits failed login attempts per IP.
// It's an in-memory rate limiter suitable for single-instance deployments.
// For multi-instance deployments, extend with Redis-backed storage.
type LoginRateLimiter struct {
	mu           sync.Mutex
	attempts     map[string]*loginAttempt
	maxPerWindow int
	windowDur    time.Duration
	lockoutDur   time.Duration
	cleanupTick  time.Duration
}

type loginAttempt struct {
	count       int
	windowStart time.Time
	lockedUntil time.Time
}

// NewLoginRateLimiter creates a new login rate limiter.
// maxAttempts: max requests per window before lockout.
// window: the time window for counting attempts.
// lockout: how long to lock out after exceeding max attempts.
func NewLoginRateLimiter(maxAttempts int, window, lockout time.Duration) *LoginRateLimiter {
	rl := &LoginRateLimiter{
		attempts:     make(map[string]*loginAttempt),
		maxPerWindow: maxAttempts,
		windowDur:    window,
		lockoutDur:   lockout,
		cleanupTick:  time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

// Allow checks whether a request from the given IP should be allowed.
// Returns nil if allowed, or an AppError if rate limited.
func (rl *LoginRateLimiter) Allow(ip string) error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	a, ok := rl.attempts[ip]
	if !ok {
		rl.attempts[ip] = &loginAttempt{
			count:       0,
			windowStart: now,
		}
		a = rl.attempts[ip]
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

// Reset clears the rate limit for a given IP.
func (rl *LoginRateLimiter) Reset(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

// Middleware creates an HTTP middleware that rate-limits requests by IP.
func (rl *LoginRateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := ClientIP(r)

		// Try to extract account from JSON body for rate limit key
		key := ip
		if r.Body != nil {
			data, _ := io.ReadAll(r.Body)
			r.Body.Close()
			if len(data) > 0 {
				var body map[string]interface{}
				if json.Unmarshal(data, &body) == nil {
					if account, ok := body["account"].(string); ok && account != "" {
						key = ip + ":" + account
					}
				}
				// Restore body for the next handler
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
