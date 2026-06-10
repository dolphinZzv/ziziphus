package api

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewLoginRateLimiter(t *testing.T) {
	rl := NewLoginRateLimiter(5, time.Minute, time.Minute)
	if rl == nil {
		t.Fatal("NewLoginRateLimiter returned nil")
	}
	if rl.maxPerWindow != 5 {
		t.Errorf("maxPerWindow = %d, want 5", rl.maxPerWindow)
	}
	if rl.windowDur != time.Minute {
		t.Errorf("windowDur = %v, want 1m", rl.windowDur)
	}
}

func TestAllow_FirstAttempt(t *testing.T) {
	rl := NewLoginRateLimiter(3, time.Minute, time.Minute)
	err := rl.Allow("192.168.1.1")
	if err != nil {
		t.Errorf("Allow() returned error for first attempt: %v", err)
	}
}

func TestAllow_WithinLimit(t *testing.T) {
	rl := NewLoginRateLimiter(3, time.Minute, time.Minute)
	for i := 0; i < 3; i++ {
		if err := rl.Allow("10.0.0.1"); err != nil {
			t.Fatalf("attempt %d: Allow() returned unexpected error: %v", i+1, err)
		}
	}
}

func TestAllow_ExceedsLimit(t *testing.T) {
	rl := NewLoginRateLimiter(2, time.Minute, time.Minute)
	_ = rl.Allow("10.0.0.1")
	_ = rl.Allow("10.0.0.1")
	err := rl.Allow("10.0.0.1")
	if err == nil {
		t.Fatal("expected rate limit error on third attempt")
	}
	if err != model.ErrRateLimited {
		t.Errorf("error = %v, want ErrRateLimited", err)
	}
}

func TestAllow_DifferentKeysIndependent(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("first IP attempt: %v", err)
	}
	if err := rl.Allow("10.0.0.2"); err != nil { // different IP
		t.Errorf("different IP should be independent: %v", err)
	}
	if err := rl.Allow("10.0.0.1"); err == nil { // same IP
		t.Errorf("same IP should be rate limited after exceeding max")
	}
}

func TestAllow_WindowReset(t *testing.T) {
	rl := NewLoginRateLimiter(1, 50*time.Millisecond, 0)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Fatalf("first attempt: %v", err)
	}
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected rate limit after first attempt")
	}
	time.Sleep(60 * time.Millisecond)
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after window reset: %v", err)
	}
}

func TestReset(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute)
	_ = rl.Allow("10.0.0.1")
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected rate limit before reset")
	}
	rl.Reset("10.0.0.1")
	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after reset: %v", err)
	}
}

func TestClientIP_IgnoresXForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.1:56789"
	r.Header.Set("X-Forwarded-For", "203.0.113.1")
	// ClientIP must NOT trust X-Forwarded-For (spoofing risk); it always uses RemoteAddr.
	if got := ClientIP(r); got != "10.0.0.1" {
		t.Errorf("ClientIP = %q, want %q (should not use X-Forwarded-For)", got, "10.0.0.1")
	}
}

func TestClientIP_RemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100:34567"
	if got := ClientIP(r); got != "192.168.1.100" {
		t.Errorf("ClientIP = %q, want 192.168.1.100", got)
	}
}

func TestClientIP_NoPortInRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "192.168.1.100"
	if got := ClientIP(r); got != "192.168.1.100" {
		t.Errorf("ClientIP = %q, want 192.168.1.100", got)
	}
}

func TestMiddleware_PassesThroughWhenNoAccount(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute)
	called := false
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMiddleware_RateLimits(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute)
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request - should pass
	w1 := httptest.NewRecorder()
	r1 := httptest.NewRequest("POST", "/login?account=alice", nil)
	r1.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w1, r1)
	if w1.Code != http.StatusOK {
		t.Errorf("first request status = %d, want %d", w1.Code, http.StatusOK)
	}

	// Second request - should be rate limited
	w2 := httptest.NewRecorder()
	r2 := httptest.NewRequest("POST", "/login?account=alice", nil)
	r2.RemoteAddr = "10.0.0.1:12345"
	handler.ServeHTTP(w2, r2)
	if w2.Code != http.StatusTooManyRequests {
		t.Errorf("rate-limited request status = %d, want %d", w2.Code, http.StatusTooManyRequests)
	}
}

func TestMiddleware_AccountFromForm(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Minute, time.Minute)
	called := false
	handler := rl.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/login", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// Note: r.FormValue("account") requires parsing the form body, which is empty here.
	// The middleware falls through to URL query.
	handler.ServeHTTP(w, r)

	if !called {
		t.Error("handler was not called when account not determinable")
	}
}

func TestConcurrent_Allow_RaceFree(t *testing.T) {
	rl := NewLoginRateLimiter(5, time.Minute, time.Minute)
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = rl.Allow("10.0.0.1")
		}()
	}
	wg.Wait()
}

func TestAllow_LockoutDuration(t *testing.T) {
	rl := NewLoginRateLimiter(1, time.Hour, 50*time.Millisecond)
	_ = rl.Allow("10.0.0.1")
	_ = rl.Allow("10.0.0.1") // rate limited, locked

	// Should still be locked out
	if err := rl.Allow("10.0.0.1"); err == nil {
		t.Fatal("expected lockout to still be in effect")
	}

	// Wait for lockout to expire
	time.Sleep(60 * time.Millisecond)

	if err := rl.Allow("10.0.0.1"); err != nil {
		t.Errorf("after lockout expiry: %v", err)
	}
}
