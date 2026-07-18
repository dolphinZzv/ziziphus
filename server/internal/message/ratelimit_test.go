package message

import (
	"context"
	"sync"
	"testing"
	"time"

	"ziziphus/pkg/model"
)

func TestNewRateLimiter(t *testing.T) {
	t.Run("typical values", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 1024)
		if rl == nil {
			t.Fatal("expected non-nil RateLimiter")
		}
		if rl.msgPerSec != 10 {
			t.Errorf("msgPerSec = %d, want 10", rl.msgPerSec)
		}
		if rl.burstSize != 5 {
			t.Errorf("burstSize = %d, want 5", rl.burstSize)
		}
		if rl.maxBodyBytes != 1024 {
			t.Errorf("maxBodyBytes = %d, want 1024", rl.maxBodyBytes)
		}
		if rl.userBuckets == nil {
			t.Error("userBuckets map should be initialized")
		}
	})

	t.Run("zero values", func(t *testing.T) {
		rl := NewRateLimiter(0, 0, 0)
		if rl.msgPerSec != 0 {
			t.Errorf("msgPerSec = %d, want 0", rl.msgPerSec)
		}
		if rl.burstSize != 0 {
			t.Errorf("burstSize = %d, want 0", rl.burstSize)
		}
		if rl.maxBodyBytes != 0 {
			t.Errorf("maxBodyBytes = %d, want 0", rl.maxBodyBytes)
		}
	})

	t.Run("negative values", func(t *testing.T) {
		rl := NewRateLimiter(-10, -5, -100)
		if rl.msgPerSec != -10 {
			t.Errorf("msgPerSec = %d, want -10", rl.msgPerSec)
		}
		if rl.burstSize != -5 {
			t.Errorf("burstSize = %d, want -5", rl.burstSize)
		}
		if rl.maxBodyBytes != -100 {
			t.Errorf("maxBodyBytes = %d, want -100", rl.maxBodyBytes)
		}
	})

	t.Run("no body limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 0)
		if rl.maxBodyBytes != 0 {
			t.Errorf("maxBodyBytes = %d, want 0", rl.maxBodyBytes)
		}
	})
}

func TestCheck_AllowsWithinRateLimit(t *testing.T) {
	burst := 5
	rl := NewRateLimiter(10, burst, 1024)
	ctx := context.Background()

	for i := 0; i < burst; i++ {
		err := rl.Check(ctx, "user1")
		if err != nil {
			t.Fatalf("call %d: expected nil, got %v", i+1, err)
		}
	}
}

func TestCheck_RejectsBeyondRateLimit(t *testing.T) {
	burst := 3
	rl := NewRateLimiter(1, burst, 1024)
	ctx := context.Background()

	for i := 0; i < burst; i++ {
		err := rl.Check(ctx, "user1")
		if err != nil {
			t.Fatalf("call %d: expected nil, got %v", i+1, err)
		}
	}

	// The next immediate call must exceed the burst and be rejected.
	err := rl.Check(ctx, "user1")
	if err == nil {
		t.Fatal("expected ErrRateLimited, got nil")
	}
	if err != model.ErrRateLimited {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}
}

func TestCheckBodySize(t *testing.T) {
	t.Run("body within limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 100)
		err := rl.CheckBodySize("hello")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
	})

	t.Run("body equals limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 100)
		err := rl.CheckBodySize(string(make([]byte, 100)))
		if err != nil {
			t.Errorf("expected nil when body size equals limit, got %v", err)
		}
	})

	t.Run("body exceeds limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 100)
		err := rl.CheckBodySize(string(make([]byte, 101)))
		if err == nil {
			t.Fatal("expected ErrMsgTooLarge, got nil")
		}
		if err != model.ErrMsgTooLarge {
			t.Fatalf("expected ErrMsgTooLarge, got %v", err)
		}
	})

	t.Run("no limit set", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 0)
		err := rl.CheckBodySize(string(make([]byte, 10000)))
		if err != nil {
			t.Errorf("expected nil when limit is zero, got %v", err)
		}
	})

	t.Run("negative limit", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, -1)
		err := rl.CheckBodySize(string(make([]byte, 10000)))
		if err != nil {
			t.Errorf("expected nil when limit is negative, got %v", err)
		}
	})

	t.Run("empty string", func(t *testing.T) {
		rl := NewRateLimiter(10, 5, 100)
		err := rl.CheckBodySize("")
		if err != nil {
			t.Errorf("expected nil for empty string, got %v", err)
		}
	})
}

func TestCheck_RateLimitResetsAfterTime(t *testing.T) {
	// Use a high refill rate so tokens recover quickly.
	msgPerSec := 100 // 100 tokens/sec => 1 token per 10 ms.
	burst := 1
	rl := NewRateLimiter(msgPerSec, burst, 1024)
	ctx := context.Background()

	// Consume the only burst token.
	if err := rl.Check(ctx, "user1"); err != nil {
		t.Fatal("expected first call to succeed")
	}

	// Immediate next call must be rate-limited.
	if err := rl.Check(ctx, "user1"); err != model.ErrRateLimited {
		t.Fatalf("expected ErrRateLimited immediately, got %v", err)
	}

	// Wait longer than one token's worth of refill time.
	time.Sleep(50 * time.Millisecond)

	// The bucket should have recovered at least 1 token.
	if err := rl.Check(ctx, "user1"); err != nil {
		t.Fatalf("expected nil after refill, got %v", err)
	}
}

func TestCheck_RateLimitRefillsPartialTokens(t *testing.T) {
	// With msgPerSec=10, each 100ms adds 1 token.
	// With burst=2, consume both, then wait 150ms (should get 1 token back,
	// not 2, because int(0.15*10)=1).
	msgPerSec := 10
	burst := 2
	rl := NewRateLimiter(msgPerSec, burst, 1024)
	ctx := context.Background()

	// Consume both tokens.
	if err := rl.Check(ctx, "user1"); err != nil {
		t.Fatal("expected call 1 to succeed")
	}
	if err := rl.Check(ctx, "user1"); err != nil {
		t.Fatal("expected call 2 to succeed")
	}

	// Third immediate call should fail.
	if err := rl.Check(ctx, "user1"); err != model.ErrRateLimited {
		t.Fatalf("expected ErrRateLimited, got %v", err)
	}

	// Wait 150ms — enough for 1 token, not enough for 2.
	time.Sleep(150 * time.Millisecond)

	// Exactly 1 token should be available.
	if err := rl.Check(ctx, "user1"); err != nil {
		t.Fatalf("expected nil after partial refill, got %v", err)
	}

	// Consume the token — now back to 0, so the next immediate call fails.
	if err := rl.Check(ctx, "user1"); err != model.ErrRateLimited {
		t.Fatalf("expected ErrRateLimited after consuming refilled token, got %v", err)
	}
}

func TestCheck_BurstCappedAtMax(t *testing.T) {
	// Tokens should never exceed burstSize.
	msgPerSec := 100
	burst := 5
	rl := NewRateLimiter(msgPerSec, burst, 1024)
	ctx := context.Background()

	// Wait more than enough time to accumulate many tokens.
	time.Sleep(200 * time.Millisecond)

	// Consume burst+1 times.  The first burst calls should succeed,
	// the (burst+1)th should fail because the bucket is capped at burst.
	for i := 0; i < burst; i++ {
		if err := rl.Check(ctx, "user1"); err != nil {
			t.Fatalf("call %d: expected nil, got %v", i+1, err)
		}
	}

	// The extra call should be rejected — the bucket was capped at burst,
	// so accumulated time does not grant more than burst tokens.
	if err := rl.Check(ctx, "user1"); err != model.ErrRateLimited {
		t.Fatalf("expected ErrRateLimited after burst exhausted, got %v", err)
	}
}

func TestCheck_ConcurrentAccessIsSafe(t *testing.T) {
	// Use msgPerSec=0 so no tokens are refilled during the test,
	// making the outcome completely deterministic.
	burst := 5
	rl := NewRateLimiter(0, burst, 1024)
	ctx := context.Background()

	var wg sync.WaitGroup
	var mu sync.Mutex
	successCount := 0
	failCount := 0
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := rl.Check(ctx, "user1")
			mu.Lock()
			switch err {
			case nil:
				successCount++
			case model.ErrRateLimited:
				failCount++
			default:
				t.Errorf("unexpected error: %v", err)
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	total := successCount + failCount
	if total != numGoroutines {
		t.Errorf("total calls accounted = %d, want %d", total, numGoroutines)
	}

	// With msgPerSec=0 exactly burst tokens exist; no more, no less.
	if successCount != burst {
		t.Errorf("successes = %d, want %d (burst)", successCount, burst)
	}
	if failCount != numGoroutines-burst {
		t.Errorf("failures = %d, want %d", failCount, numGoroutines-burst)
	}
}

func TestCheck_DifferentUsersIndependent(t *testing.T) {
	burst := 2
	rl := NewRateLimiter(1, burst, 1024)
	ctx := context.Background()

	// User A: consume all burst tokens.
	for i := 0; i < burst; i++ {
		if err := rl.Check(ctx, "userA"); err != nil {
			t.Fatalf("userA call %d: expected nil, got %v", i+1, err)
		}
	}

	// User A: next immediate call must be rejected.
	if err := rl.Check(ctx, "userA"); err != model.ErrRateLimited {
		t.Fatalf("userA: expected ErrRateLimited, got %v", err)
	}

	// User B: should still enjoy a full burst.
	for i := 0; i < burst; i++ {
		if err := rl.Check(ctx, "userB"); err != nil {
			t.Fatalf("userB call %d: expected nil, got %v", i+1, err)
		}
	}

	// User B: exhausted — next call fails.
	if err := rl.Check(ctx, "userB"); err != model.ErrRateLimited {
		t.Fatalf("userB: expected ErrRateLimited, got %v", err)
	}

	// User A: still rate-limited independently (no meaningful time elapsed).
	if err := rl.Check(ctx, "userA"); err != model.ErrRateLimited {
		t.Fatalf("userA: expected ErrRateLimited after userB tests, got %v", err)
	}
}

func TestCheck_MultipleUsersRoundRobin(t *testing.T) {
	burst := 1
	rl := NewRateLimiter(1, burst, 1024)
	ctx := context.Background()

	// user1 succeeds, user2 succeeds, user3 succeeds — each has their own bucket.
	users := []string{"user1", "user2", "user3"}
	for _, u := range users {
		if err := rl.Check(ctx, u); err != nil {
			t.Fatalf("user %s: expected nil, got %v", u, err)
		}
	}

	// All three users are now at 0 tokens.
	for _, u := range users {
		if err := rl.Check(ctx, u); err != model.ErrRateLimited {
			t.Fatalf("user %s: expected ErrRateLimited, got %v", u, err)
		}
	}
}

func TestCleanup_RemovesStaleEntries(t *testing.T) {
	rl := NewRateLimiter(10, 5, 1024)
	ctx := context.Background()

	rl.Check(ctx, "user_stale")
	rl.Check(ctx, "user_fresh")

	rl.mu.Lock()
	rl.userBuckets["user_stale"] = &bucket{
		tokens:    5,
		lastCheck: time.Now().Add(-20 * time.Minute),
	}
	rl.userBuckets["user_fresh"] = &bucket{
		tokens:    5,
		lastCheck: time.Now().Add(-1 * time.Minute),
	}
	rl.mu.Unlock()

	rl.cleanup()

	rl.mu.Lock()
	_, staleExists := rl.userBuckets["user_stale"]
	_, freshExists := rl.userBuckets["user_fresh"]
	rl.mu.Unlock()

	if staleExists {
		t.Error("stale user should have been removed")
	}
	if !freshExists {
		t.Error("fresh user should still be present")
	}
}

func TestCleanup_EmptyBuckets_NoPanic(t *testing.T) {
	rl := NewRateLimiter(10, 5, 1024)
	rl.cleanup()
}

func TestRateLimiter_Stop(t *testing.T) {
	rl := NewRateLimiter(10, 5, 1024)

	// Stop should not panic and should close the stopped channel.
	done := make(chan struct{})
	go func() {
		rl.Stop()
		close(done)
	}()

	select {
	case <-done:
		// OK — Stop returned without blocking
	case <-time.After(2 * time.Second):
		t.Fatal("Stop() did not return within 2 seconds")
	}

	// Verify the cleanup goroutine has exited by checking
	// that the stopped channel is closed.
	select {
	case _, ok := <-rl.stopped:
		if ok {
			t.Error("stopped channel should be closed after Stop()")
		}
	default:
		t.Error("stopped channel should be closed")
	}
}

func TestCheck_ContextPassed(t *testing.T) {
	// The context parameter is accepted for API consistency.
	// RateLimiter does not use it for cancellation, but we verify
	// that passing a valid context works fine and passing nil also
	// does not panic (since ctx is never dereferenced).
	rl := NewRateLimiter(10, 5, 1024)

	if err := rl.Check(context.Background(), "user1"); err != nil {
		t.Fatalf("expected nil with context.Background(), got %v", err)
	}
}
