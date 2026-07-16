package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"ziziphus/pkg/model"
)

// setupRedis creates a miniredis instance and returns a redis client connected to it.
func setupRedis(t *testing.T) (*miniredis.Miniredis, *redis.Client) {
	t.Helper()
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatalf("miniredis.Start: %v", err)
	}
	t.Cleanup(mr.Close)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })
	return mr, rdb
}

// ---------------------------------------------------------------------------
// LoginRateLimiter — Redis path
// ---------------------------------------------------------------------------

func TestLoginRateLimiter_Redis_FirstAttempt(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(5, time.Minute, time.Minute, rdb)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("first attempt should be allowed: %v", err)
	}
}

func TestLoginRateLimiter_Redis_ExceedsLimit(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(2, time.Minute, time.Minute, rdb)
	for i := 0; i < 2; i++ {
		if err := rl.Allow("10.0.0.1"); err != nil {
			t.Fatalf("attempt %d should be allowed: %v", i+1, err)
		}
	}
	// Third attempt — rate limited
	if err := rl.Allow("10.0.0.1"); err != model.ErrRateLimited {
		t.Errorf("third attempt should be rate limited, got: %v", err)
	}
}

func TestLoginRateLimiter_Redis_LockoutDuration(t *testing.T) {
	mr, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, time.Hour, 2*time.Second, rdb)
	_ = rl.Allow("10.0.0.1")
	_ = rl.Allow("10.0.0.1") // triggers lockout

	// Still locked out
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected lockout to still be in effect")
	}

	// Advance past lockout
	mr.FastForward(2100 * time.Millisecond)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after lockout expiry: %v", err)
	}
}

func TestLoginRateLimiter_Redis_WindowReset(t *testing.T) {
	mr, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, 2*time.Second, 0, rdb)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected rate limit after first attempt")
	}

	// Advance miniredis clock past the window
	mr.FastForward(2100 * time.Millisecond)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after window reset: %v", err)
	}
}

func TestLoginRateLimiter_Redis_DifferentKeys(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute, rdb)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("first IP: %v", err)
	}
	// Different IP should be independent
	if err := rl.Allow("10.0.0.2"); err != nil {
		t.Errorf("different IP should be independent: %v", err)
	}
	// Same IP again — should be limited
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Errorf("same IP should be rate limited")
	}
}

func TestLoginRateLimiter_Redis_Reset(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute, rdb)

	_ = rl.Allow("10.0.0.1")
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected rate limit before reset")
	}

	rl.Reset("10.0.0.1")

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after reset: %v", err)
	}
}

func TestLoginRateLimiter_Redis_FailOpen(t *testing.T) {
	// Use a stopped Redis to simulate failure
	mr, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute, rdb)
	mr.Close() // kill Redis

	// Should NOT block requests when Redis is down (fail-open)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("should fail open on Redis error: %v", err)
	}
}

func TestLoginRateLimiter_Redis_Middleware(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute, rdb)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request — should pass
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/login", nil)
	r1.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("first request status = %d, want %d", w1.Code, http.StatusOK)
	}

	// Second request — rate limited
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/login", nil)
	r2.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second request status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

// ---------------------------------------------------------------------------
// RegisterLimiter — Redis path
// ---------------------------------------------------------------------------

func TestRegisterLimiter_Redis_FirstAttempt(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewRegisterLimiter(5, time.Hour, rdb)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("first attempt should be allowed: %v", err)
	}
}

func TestRegisterLimiter_Redis_ExceedsLimit(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewRegisterLimiter(3, time.Minute, rdb)
	for i := 0; i < 3; i++ {
		if err := rl.Allow("10.0.0.1"); err != nil {
			t.Fatalf("attempt %d: %v", i+1, err)
		}
	}
	if err := rl.Allow("10.0.0.1"); err != model.ErrRateLimited {
		t.Errorf("should be rate limited, got: %v", err)
	}
}

func TestRegisterLimiter_Redis_WindowReset(t *testing.T) {
	mr, rdb := setupRedis(t)
	rl := NewRegisterLimiter(1, 2*time.Second, rdb)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("first: %v", err)
	}
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected rate limit")
	}

	mr.FastForward(2100 * time.Millisecond)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after window reset: %v", err)
	}
}

func TestRegisterLimiter_Redis_FailOpen(t *testing.T) {
	mr, rdb := setupRedis(t)
	rl := NewRegisterLimiter(1, time.Minute, rdb)
	mr.Close()

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("should fail open: %v", err)
	}
}

func TestRegisterLimiter_Redis_Middleware(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewRegisterLimiter(1, time.Minute, rdb)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/api/v1/users/register", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("first status = %d, want %d", w.Code, http.StatusOK)
	}

	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/api/v1/users/register", nil)
	r2.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("second status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

// ---------------------------------------------------------------------------
// GlobalRateLimiter — Redis path
// ---------------------------------------------------------------------------

func TestGlobalRateLimiter_Redis_FirstAttempt(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewGlobalRateLimiter(10, 20, rdb)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("first attempt should be allowed: %v", err)
	}
}

func TestGlobalRateLimiter_Redis_ExceedsBurst(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewGlobalRateLimiter(1, 2, rdb) // 1 token/s, burst 2

	// Consume burst
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("burst 1: %v", err)
	}
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("burst 2: %v", err)
	}
	// Third in same second — should be limited
	if err := rl.Allow("10.0.0.1"); err != model.ErrRateLimited {
		t.Errorf("should be rate limited, got: %v", err)
	}
}

func TestGlobalRateLimiter_Redis_FailOpen(t *testing.T) {
	mr, rdb := setupRedis(t)
	rl := NewGlobalRateLimiter(10, 20, rdb)
	mr.Close()

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("should fail open: %v", err)
	}
}

func TestGlobalRateLimiter_Redis_MiddlewareSkipsHealth(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewGlobalRateLimiter(1, 1, rdb)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("health should always pass: %d", w.Code)
	}
}

func TestGlobalRateLimiter_Redis_DifferentIPs(t *testing.T) {
	_, rdb := setupRedis(t)
	rl := NewGlobalRateLimiter(1, 1, rdb)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("IP 1: %v", err)
	}
	if err := rl.Allow("10.0.0.2"); err != nil {
		t.Errorf("IP 2 should be independent: %v", err)
	}
}
