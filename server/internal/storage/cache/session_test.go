package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"siciv.space/agent/panda_ai/pkg/model"
)

func setupSessionCache(t *testing.T) (*miniredis.Miniredis, *SessionCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := NewRedisClient(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	return mr, NewSessionCache(client)
}

func TestNewSessionCache(t *testing.T) {
	_, sc := setupSessionCache(t)
	if sc == nil {
		t.Fatal("NewSessionCache returned nil")
	}
}

func TestSessionCache_SetAndGet(t *testing.T) {
	_, sc := setupSessionCache(t)
	ctx := context.Background()

	s := &model.Session{
		SessionID:  "sess_1",
		UserID:     "user_1",
		Device:     1,
		DeviceName: "ios",
		Status:     1,
		LoginAt:    1000,
		LastActive: 2000,
	}

	err := sc.Set(ctx, s)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	got, err := sc.Get(ctx, "sess_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.SessionID != "sess_1" {
		t.Errorf("SessionID = %q, want sess_1", got.SessionID)
	}
	if got.UserID != "user_1" {
		t.Errorf("UserID = %q, want user_1", got.UserID)
	}
	if got.Device != 1 {
		t.Errorf("Device = %d, want 1", got.Device)
	}
	if got.DeviceName != "ios" {
		t.Errorf("DeviceName = %q, want ios", got.DeviceName)
	}
	if got.Status != 1 {
		t.Errorf("Status = %d, want 1", got.Status)
	}
}

func TestSessionCache_Get_NotFound(t *testing.T) {
	_, sc := setupSessionCache(t)
	ctx := context.Background()

	_, err := sc.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for non-existent session")
	}
}

func TestSessionCache_Delete(t *testing.T) {
	mr, sc := setupSessionCache(t)
	ctx := context.Background()

	s := &model.Session{
		SessionID: "sess_del",
		UserID:    "user_del",
	}
	err := sc.Set(ctx, s)
	if err != nil {
		t.Fatalf("Set: %v", err)
	}

	err = sc.Delete(ctx, "sess_del", "user_del")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	// Verify session key is gone
	_, err = sc.Get(ctx, "sess_del")
	if err == nil {
		t.Error("expected session to be deleted")
	}

	// Verify user sessions set
	if mr.Exists("user:sessions:user_del") {
		t.Error("user sessions set should be empty or deleted")
	}
}

func TestSessionCache_GetUserSessionIDs(t *testing.T) {
	_, sc := setupSessionCache(t)
	ctx := context.Background()

	// Add two sessions for the same user
	s1 := &model.Session{SessionID: "sess_a", UserID: "user_x"}
	s2 := &model.Session{SessionID: "sess_b", UserID: "user_x"}
	_ = sc.Set(ctx, s1)
	_ = sc.Set(ctx, s2)

	ids, err := sc.GetUserSessionIDs(ctx, "user_x")
	if err != nil {
		t.Fatalf("GetUserSessionIDs: %v", err)
	}
	if len(ids) != 2 {
		t.Fatalf("got %d session IDs, want 2", len(ids))
	}

	seen := make(map[string]bool)
	for _, id := range ids {
		seen[id] = true
	}
	if !seen["sess_a"] {
		t.Error("sess_a not in user sessions")
	}
	if !seen["sess_b"] {
		t.Error("sess_b not in user sessions")
	}
}

func TestSessionCache_GetUserSessionIDs_Empty(t *testing.T) {
	_, sc := setupSessionCache(t)
	ctx := context.Background()

	ids, err := sc.GetUserSessionIDs(ctx, "user_no_sessions")
	if err != nil {
		t.Fatalf("GetUserSessionIDs: %v", err)
	}
	if len(ids) != 0 {
		t.Errorf("got %d session IDs, want 0", len(ids))
	}
}

func TestSessionCache_SetConnSession(t *testing.T) {
	sc := setupSessionCacheSimple(t)
	ctx := context.Background()

	err := sc.SetConnSession(ctx, "conn_1", "sess_1")
	if err != nil {
		t.Fatalf("SetConnSession: %v", err)
	}

	sessionID, err := sc.GetConnSession(ctx, "conn_1")
	if err != nil {
		t.Fatalf("GetConnSession: %v", err)
	}
	if sessionID != "sess_1" {
		t.Errorf("sessionID = %q, want sess_1", sessionID)
	}
}

func TestSessionCache_GetConnSession_NotFound(t *testing.T) {
	sc := setupSessionCacheSimple(t)
	ctx := context.Background()

	_, err := sc.GetConnSession(ctx, "conn_nonexistent")
	if err == nil {
		t.Error("expected error for non-existent conn session")
	}
}

func TestSessionCache_DelConnSession(t *testing.T) {
	sc := setupSessionCacheSimple(t)
	ctx := context.Background()

	_ = sc.SetConnSession(ctx, "conn_1", "sess_1")
	err := sc.DelConnSession(ctx, "conn_1")
	if err != nil {
		t.Fatalf("DelConnSession: %v", err)
	}

	_, err = sc.GetConnSession(ctx, "conn_1")
	if err == nil {
		t.Error("expected conn session to be deleted")
	}
}

func TestSessionCache_Delete_CleansUpUserSessions(t *testing.T) {
	mr, sc := setupSessionCache(t)
	ctx := context.Background()

	s := &model.Session{SessionID: "sess_unique", UserID: "user_unique"}
	_ = sc.Set(ctx, s)

	// Before delete, user should have this session
	ids, _ := sc.GetUserSessionIDs(ctx, "user_unique")
	if len(ids) != 1 {
		t.Fatalf("expected 1 session before delete, got %d", len(ids))
	}

	_ = sc.Delete(ctx, "sess_unique", "user_unique")

	// After delete, user sessions should be empty
	if mr.Exists("user:sessions:user_unique") {
		ids, _ = sc.GetUserSessionIDs(ctx, "user_unique")
		if len(ids) != 0 {
			t.Errorf("expected 0 sessions after delete, got %d", len(ids))
		}
	}
}

func setupSessionCacheSimple(t *testing.T) *SessionCache {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := NewRedisClient(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	return NewSessionCache(client)
}

func TestSessionCache_Delete_NoSessionKey(t *testing.T) {
	sc := setupSessionCacheSimple(t)
	ctx := context.Background()

	// Should not panic if the session key doesn't exist
	err := sc.Delete(ctx, "sess_nonexistent", "user_nonexistent")
	if err != nil {
		// This is fine - the redis DEL/SREM will succeed even if keys don't exist
		t.Logf("Delete returned error (acceptable): %v", err)
	}
}
