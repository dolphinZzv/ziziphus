package gateway

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func newTestConnection(connID, userID, sessionID string) *Connection {
	return &Connection{
		ConnID:        connID,
		UserID:        userID,
		SessionID:     sessionID,
		Device:        1,
		CreatedAt:     time.Now(),
		LastHeartbeat: time.Now().UnixMilli(),
	}
}

// ---------------------------------------------------------------------------
// 1) NewManager creates an empty manager
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	m := NewManager()
	if m == nil {
		t.Fatal("NewManager() returned nil")
	}
	if count := m.Count(); count != 0 {
		t.Fatalf("expected Count() == 0, got %d", count)
	}
	if conns := m.All(); len(conns) != 0 {
		t.Fatalf("expected All() to return empty slice, got %d element(s)", len(conns))
	}
}

// ---------------------------------------------------------------------------
// 2) Add a single connection and verify all accessors
// ---------------------------------------------------------------------------

func TestAdd_SingleConnection(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	conn := newTestConnection("conn1", "user1", "session1")

	m.Add(ctx, conn)

	// Count
	if count := m.Count(); count != 1 {
		t.Fatalf("expected Count() == 1, got %d", count)
	}

	// Get
	got := m.Get(ctx, "conn1")
	if got == nil {
		t.Fatal("expected Get() to return non-nil connection")
	}
	if got.ConnID != "conn1" {
		t.Fatalf("expected ConnID 'conn1', got %q", got.ConnID)
	}

	// GetBySessionID
	sessConn := m.GetBySessionID(ctx, "session1")
	c, ok := sessConn.(*Connection)
	if !ok || c == nil {
		t.Fatal("expected GetBySessionID() to return non-nil *Connection")
	}
	if c.ConnID != "conn1" {
		t.Fatalf("expected ConnID 'conn1', got %q", c.ConnID)
	}

	// GetByUserID
	userConns := m.GetByUserID(ctx, "user1")
	if userConns == nil {
		t.Fatal("expected GetByUserID() to return non-nil")
	}
	if len(userConns) != 1 {
		t.Fatalf("expected 1 connection for user, got %d", len(userConns))
	}
	uc, ok := userConns[0].(*Connection)
	if !ok {
		t.Fatal("expected GetByUserID() element to be *Connection")
	}
	if uc.ConnID != "conn1" {
		t.Fatalf("expected ConnID 'conn1', got %q", uc.ConnID)
	}

	// All
	all := m.All()
	if len(all) != 1 {
		t.Fatalf("expected All() to have 1 element, got %d", len(all))
	}
	if all[0].ConnID != "conn1" {
		t.Fatalf("expected All()[0].ConnID == 'conn1', got %q", all[0].ConnID)
	}
}

// ---------------------------------------------------------------------------
// 3) Add multiple connections for the same user
// ---------------------------------------------------------------------------

func TestAdd_MultipleConnectionsSameUser(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	conn1 := newTestConnection("conn1", "user1", "session1")
	conn2 := newTestConnection("conn2", "user1", "session2")

	m.Add(ctx, conn1)
	m.Add(ctx, conn2)

	if count := m.Count(); count != 2 {
		t.Fatalf("expected Count() == 2, got %d", count)
	}

	userConns := m.GetByUserID(ctx, "user1")
	if userConns == nil {
		t.Fatal("expected GetByUserID() to return non-nil")
	}
	if len(userConns) != 2 {
		t.Fatalf("expected 2 connections for user, got %d", len(userConns))
	}

	ids := make(map[string]bool)
	for _, c := range userConns {
		conn, ok := c.(*Connection)
		if !ok {
			t.Fatal("expected element to be *Connection")
		}
		ids[conn.ConnID] = true
	}
	if !ids["conn1"] || !ids["conn2"] {
		t.Fatal("expected both conn1 and conn2 in GetByUserID results")
	}
}

// ---------------------------------------------------------------------------
// 4) Remove cleans up all internal maps
// ---------------------------------------------------------------------------

func TestRemove_CleansAllMaps(t *testing.T) {
	m := NewManager()
	ctx := context.Background()
	conn := newTestConnection("conn1", "user1", "session1")

	m.Add(ctx, conn)
	m.Remove(ctx, "conn1")

	if count := m.Count(); count != 0 {
		t.Fatalf("expected Count() == 0, got %d", count)
	}
	if got := m.Get(ctx, "conn1"); got != nil {
		t.Fatal("expected Get() to return nil after Remove")
	}
	if got := m.GetBySessionID(ctx, "session1"); got != nil {
		if c, ok := got.(*Connection); ok && c != nil {
			t.Fatal("expected GetBySessionID() to return nil after Remove")
		}
	}
	if got := m.GetByUserID(ctx, "user1"); got != nil {
		t.Fatal("expected GetByUserID() to return nil after Remove")
	}
}

// ---------------------------------------------------------------------------
// 5) Remove non-existent connection does not panic
// ---------------------------------------------------------------------------

func TestRemove_NonExistent_NoPanic(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	// Should not panic
	m.Remove(ctx, "non-existent")

	if count := m.Count(); count != 0 {
		t.Fatalf("expected Count() == 0 after removing non-existent, got %d", count)
	}
}

// ---------------------------------------------------------------------------
// 6) Get returns nil for non-existent connID
// ---------------------------------------------------------------------------

func TestGet_NonExistent_ReturnsNil(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	if got := m.Get(ctx, "non-existent"); got != nil {
		t.Fatal("expected Get() to return nil")
	}
}

// ---------------------------------------------------------------------------
// 7) GetBySessionID returns nil for non-existent sessionID
// ---------------------------------------------------------------------------

func TestGetBySessionID_NonExistent_ReturnsNil(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	got := m.GetBySessionID(ctx, "non-existent")
	c, _ := got.(*Connection)
	if c != nil {
		t.Fatal("expected GetBySessionID() to return nil")
	}
}

// ---------------------------------------------------------------------------
// 8) GetByUserID returns nil for non-existent userID
// ---------------------------------------------------------------------------

func TestGetByUserID_NonExistent_ReturnsNil(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	if got := m.GetByUserID(ctx, "non-existent"); got != nil {
		t.Fatal("expected GetByUserID() to return nil")
	}
}

// ---------------------------------------------------------------------------
// 9) All returns all connections
// ---------------------------------------------------------------------------

func TestAll_ReturnsAllConnections(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	const n = 3
	for i := 0; i < n; i++ {
		conn := newTestConnection(
			fmt.Sprintf("conn%d", i),
			fmt.Sprintf("user%d", i),
			fmt.Sprintf("session%d", i),
		)
		m.Add(ctx, conn)
	}

	all := m.All()
	if len(all) != n {
		t.Fatalf("expected All() to return %d connections, got %d", n, len(all))
	}

	ids := make(map[string]bool)
	for _, c := range all {
		ids[c.ConnID] = true
	}
	for i := 0; i < n; i++ {
		key := fmt.Sprintf("conn%d", i)
		if !ids[key] {
			t.Fatalf("expected All() to include %q", key)
		}
	}
}

// ---------------------------------------------------------------------------
// 10) Concurrent Add / Remove is race-free
//
// Run with: go test -race -run TestConcurrent_AddRemove_RaceFree
// ---------------------------------------------------------------------------

func TestConcurrent_AddRemove_RaceFree(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	const n = 100

	// Concurrently add n connections (10 users, 10 conns each)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			conn := newTestConnection(
				fmt.Sprintf("conn%d", idx),
				fmt.Sprintf("user%d", idx%10),
				fmt.Sprintf("session%d", idx),
			)
			m.Add(ctx, conn)
		}(i)
	}
	wg.Wait()

	if count := m.Count(); count != n {
		t.Fatalf("expected Count() == %d after add phase, got %d", n, count)
	}

	// Concurrent reads
	var wg2 sync.WaitGroup
	for i := 0; i < n; i++ {
		wg2.Add(1)
		go func(idx int) {
			defer wg2.Done()
			_ = m.Get(ctx, fmt.Sprintf("conn%d", idx))
			_ = m.GetBySessionID(ctx, fmt.Sprintf("session%d", idx))
			_ = m.GetByUserID(ctx, fmt.Sprintf("user%d", idx%10))
		}(i)
	}
	wg2.Wait()

	// Concurrently remove first half of connections
	var wg3 sync.WaitGroup
	for i := 0; i < n/2; i++ {
		wg3.Add(1)
		go func(idx int) {
			defer wg3.Done()
			m.Remove(ctx, fmt.Sprintf("conn%d", idx))
		}(i)
	}
	wg3.Wait()

	expected := n - n/2
	if count := m.Count(); count != expected {
		t.Fatalf("expected Count() == %d after remove phase, got %d", expected, count)
	}
}

// ---------------------------------------------------------------------------
// 11) Add / Remove updates Count correctly
// ---------------------------------------------------------------------------

func TestAddRemove_CountUpdates(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	// Incrementally add
	for i := 1; i <= 5; i++ {
		conn := newTestConnection(
			fmt.Sprintf("conn%d", i),
			"user1",
			fmt.Sprintf("session%d", i),
		)
		m.Add(ctx, conn)
		if count := m.Count(); count != i {
			t.Fatalf("after adding %d conns, expected Count() == %d, got %d", i, i, count)
		}
	}

	// Incrementally remove
	for i := 5; i >= 1; i-- {
		m.Remove(ctx, fmt.Sprintf("conn%d", i))
		expected := i - 1
		if count := m.Count(); count != expected {
			t.Fatalf("after removing down to %d, expected Count() == %d, got %d", expected, expected, count)
		}
	}
}

// ---------------------------------------------------------------------------
// 12) Remove last connection for a user cleans up the userConns entry
// ---------------------------------------------------------------------------

func TestRemove_LastConnectionForUser_CleansUserConnsEntry(t *testing.T) {
	m := NewManager()
	ctx := context.Background()

	conn1 := newTestConnection("conn1", "user1", "session1")
	conn2 := newTestConnection("conn2", "user2", "session2")

	m.Add(ctx, conn1)
	m.Add(ctx, conn2)

	// Remove user1's only connection
	m.Remove(ctx, "conn1")

	// GetByUserID for user1 should now return nil
	if got := m.GetByUserID(ctx, "user1"); got != nil {
		t.Fatal("expected GetByUserID('user1') to return nil after last connection removed")
	}

	// user2 should still be accessible
	if got := m.GetByUserID(ctx, "user2"); got == nil {
		t.Fatal("expected GetByUserID('user2') to be non-nil")
	}

	// Count should be 1
	if count := m.Count(); count != 1 {
		t.Fatalf("expected Count() == 1, got %d", count)
	}

	// Remove user2's only connection as well
	m.Remove(ctx, "conn2")

	if got := m.GetByUserID(ctx, "user2"); got != nil {
		t.Fatal("expected GetByUserID('user2') to return nil after last connection removed")
	}
	if count := m.Count(); count != 0 {
		t.Fatalf("expected Count() == 0, got %d", count)
	}
}
