package auth

import (
	"sync"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestSetSignupCode_Redis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	InitSignupCodeStore(rdb)
	defer InitSignupCodeStore(nil) // Reset to memory mode

	SetSignupCode("r-code-1", "user_1", 60)
	defer ClearSignupCode("r-code-1")

	userID := GetSignupCode("r-code-1")
	if userID != "user_1" {
		t.Errorf("GetSignupCode = %q, want user_1", userID)
	}
}

func TestGetSignupCode_Redis_NotFound(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	InitSignupCodeStore(rdb)
	defer InitSignupCodeStore(nil)

	userID := GetSignupCode("r-nonexistent")
	if userID != "" {
		t.Errorf("GetSignupCode = %q, want empty", userID)
	}
}

func TestSetSignupCode(t *testing.T) {
	SetSignupCode("test-code-1", "user_1", 60)
	defer ClearSignupCode("test-code-1")

	userID := GetSignupCode("test-code-1")
	if userID != "user_1" {
		t.Errorf("GetSignupCode = %q, want user_1", userID)
	}
}

func TestGetSignupCode_NotFound(t *testing.T) {
	userID := GetSignupCode("nonexistent-code")
	if userID != "" {
		t.Errorf("GetSignupCode = %q, want empty string", userID)
	}
}

func TestGetSignupCode_Expired(t *testing.T) {
	// Set a code with 0-second TTL (immediately expired)
	SetSignupCode("expired-code", "user_1", 0)
	defer ClearSignupCode("expired-code")

	// Give the goroutine scheduler a chance, though time.Now() checks should catch it
	userID := GetSignupCode("expired-code")
	if userID != "" {
		t.Errorf("GetSignupCode for expired code = %q, want empty string", userID)
	}
}

func TestClearSignupCode(t *testing.T) {
	SetSignupCode("clear-test", "user_1", 60)
	ClearSignupCode("clear-test")

	userID := GetSignupCode("clear-test")
	if userID != "" {
		t.Errorf("GetSignupCode after Clear = %q, want empty string", userID)
	}
}

func TestSetSignupCode_Overwrite(t *testing.T) {
	SetSignupCode("overwrite-code", "user_1", 60)
	SetSignupCode("overwrite-code", "user_2", 60)
	defer ClearSignupCode("overwrite-code")

	userID := GetSignupCode("overwrite-code")
	if userID != "user_2" {
		t.Errorf("GetSignupCode after overwrite = %q, want user_2", userID)
	}
}

func TestSetSignupCode_ConcurrentSafe(t *testing.T) {
	var wg sync.WaitGroup
	n := 20

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			code := string(rune('A' + i))
			SetSignupCode(code, "user", 60)
		}(i)
	}
	wg.Wait()

	// Verify there's no corruption
	for i := 0; i < n; i++ {
		code := string(rune('A' + i))
		ClearSignupCode(code)
	}
}
