package auth

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func TestOAuthStateStore_Memory(t *testing.T) {
	store := NewOAuthStateStore(nil)

	store.Set("state1", &OAuthState{Provider: "github", Mode: "login"})
	result := store.GetAndClear("state1")
	if result == nil {
		t.Fatal("expected state, got nil")
	}
	if result.Provider != "github" {
		t.Errorf("Provider = %q, want github", result.Provider)
	}

	// GetAndClear should remove it
	if got := store.GetAndClear("state1"); got != nil {
		t.Error("expected nil after GetAndClear")
	}
}

func TestOAuthStateStore_Memory_Expired(t *testing.T) {
	store := NewOAuthStateStore(nil)

	store.Set("expired", &OAuthState{Provider: "github", Mode: "login"})
	// Manually expire it
	store.mu.Lock()
	store.mem["expired"].ExpiresAt = time.Now().Add(-1 * time.Minute)
	store.mu.Unlock()

	result := store.GetAndClear("expired")
	if result != nil {
		t.Error("expected nil for expired state")
	}
}

func TestOAuthStateStore_Redis(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOAuthStateStore(rdb)

	store.Set("rstate1", &OAuthState{Provider: "google", Mode: "bind", UserID: "user_1"})
	result := store.GetAndClear("rstate1")
	if result == nil {
		t.Fatal("expected state, got nil")
	}
	if result.Provider != "google" {
		t.Errorf("Provider = %q, want google", result.Provider)
	}
	if result.UserID != "user_1" {
		t.Errorf("UserID = %q, want user_1", result.UserID)
	}

	// GetAndClear should remove it
	if got := store.GetAndClear("rstate1"); got != nil {
		t.Error("expected nil after GetAndClear")
	}
}

func TestOAuthStateStore_Redis_NotFound(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOAuthStateStore(rdb)

	result := store.GetAndClear("nonexistent")
	if result != nil {
		t.Error("expected nil for nonexistent state")
	}
}

func TestOAuthStateStore_Redis_Expired(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis.Run: %v", err)
	}
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	store := NewOAuthStateStore(rdb)

	store.Set("rexpired", &OAuthState{Provider: "github", Mode: "login"})

	// Manually expire miniredis data
	mr.FastForward(11 * time.Minute)

	result := store.GetAndClear("rexpired")
	if result != nil {
		t.Error("expected nil for expired state")
	}
}
