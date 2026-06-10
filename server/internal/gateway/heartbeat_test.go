package gateway

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestDefaultHeartbeatConfig(t *testing.T) {
	cfg := DefaultHeartbeatConfig()
	if cfg.Interval != 30*time.Second {
		t.Errorf("Interval = %v, want 30s", cfg.Interval)
	}
	if cfg.Timeout != 90*time.Second {
		t.Errorf("Timeout = %v, want 90s", cfg.Timeout)
	}
}

func TestNewHeartbeat(t *testing.T) {
	mgr := NewManager()
	cfg := HeartbeatConfig{Interval: time.Second, Timeout: 3 * time.Second}
	hb := NewHeartbeat(mgr, cfg)
	if hb == nil {
		t.Fatal("NewHeartbeat returned nil")
	}
	if hb.manager != mgr {
		t.Error("manager not set")
	}
	if hb.config != cfg {
		t.Error("config not set")
	}
	if hb.stopped.Load() {
		t.Error("heartbeat should not be stopped initially")
	}
}

func TestHeartbeat_Stop(t *testing.T) {
	mgr := NewManager()
	hb := NewHeartbeat(mgr, DefaultHeartbeatConfig())
	hb.Stop()
	if !hb.stopped.Load() {
		t.Error("heartbeat should be stopped after Stop()")
	}
}

func TestHeartbeat_Start_StopsOnContextCancel(t *testing.T) {
	mgr := NewManager()
	cfg := HeartbeatConfig{Interval: 10 * time.Millisecond, Timeout: 30 * time.Millisecond}
	hb := NewHeartbeat(mgr, cfg)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		hb.Start(ctx, func(ctx context.Context, connID string) {})
		close(done)
	}()

	// Let it tick a couple times
	time.Sleep(25 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// ok
	case <-time.After(time.Second):
		t.Fatal("heartbeat did not stop after context cancel")
	}
}

func TestHeartbeat_CheckTimeouts_NoConnections(t *testing.T) {
	mgr := NewManager()
	cfg := HeartbeatConfig{Interval: time.Second, Timeout: time.Second}
	hb := NewHeartbeat(mgr, cfg)

	// Should not panic and not call onTimeout
	timeoutCalled := false
	hb.checkTimeouts(context.Background(), func(ctx context.Context, connID string) {
		timeoutCalled = true
	})
	if timeoutCalled {
		t.Error("onTimeout should not be called with no connections")
	}
}

func TestHeartbeat_CheckTimeouts_ActiveConnection(t *testing.T) {
	mgr := NewManager()
	ctx := context.Background()
	mgr.Add(ctx, &Connection{
		ConnID:        "c1",
		UserID:        "u1",
		SessionID:     "s1",
		LastHeartbeat: time.Now().UnixMilli(),
	})

	cfg := HeartbeatConfig{Interval: time.Second, Timeout: time.Minute}
	hb := NewHeartbeat(mgr, cfg)

	timeoutCalled := false
	hb.checkTimeouts(ctx, func(_ context.Context, connID string) {
		timeoutCalled = true
		t.Errorf("onTimeout called for conn %s", connID)
	})
	if timeoutCalled {
		t.Error("onTimeout should not be called for active connection")
	}
}

func TestHeartbeat_CheckTimeouts_TimeoutConnection(t *testing.T) {
	mgr := NewManager()
	mgr.Add(context.Background(), &Connection{
		ConnID:        "c_stale",
		UserID:        "u1",
		SessionID:     "s1",
		LastHeartbeat: time.Now().Add(-2 * time.Minute).UnixMilli(), // 2 min old
	})

	cfg := HeartbeatConfig{Interval: time.Second, Timeout: 30 * time.Second}
	hb := NewHeartbeat(mgr, cfg)

	var timeoutConns []string
	hb.checkTimeouts(context.Background(), func(_ context.Context, connID string) {
		timeoutConns = append(timeoutConns, connID)
	})

	if len(timeoutConns) != 1 {
		t.Fatalf("onTimeout called %d times, want 1", len(timeoutConns))
	}
	if timeoutConns[0] != "c_stale" {
		t.Errorf("timeout conn = %q, want c_stale", timeoutConns[0])
	}
}

func TestHeartbeat_CheckTimeouts_MixedConnections(t *testing.T) {
	mgr := NewManager()
	now := time.Now()
	mgr.Add(context.Background(), &Connection{
		ConnID:        "c_fresh",
		UserID:        "u1",
		LastHeartbeat: now.UnixMilli(),
	})
	mgr.Add(context.Background(), &Connection{
		ConnID:        "c_old",
		UserID:        "u1",
		LastHeartbeat: now.Add(-2 * time.Minute).UnixMilli(),
	})
	mgr.Add(context.Background(), &Connection{
		ConnID:        "c_older",
		UserID:        "u2",
		LastHeartbeat: now.Add(-5 * time.Minute).UnixMilli(),
	})

	cfg := HeartbeatConfig{Interval: time.Second, Timeout: time.Minute}
	hb := NewHeartbeat(mgr, cfg)

	var timeoutConns []string
	hb.checkTimeouts(context.Background(), func(_ context.Context, connID string) {
		timeoutConns = append(timeoutConns, connID)
	})

	if len(timeoutConns) != 2 {
		t.Fatalf("onTimeout called %d times, want 2", len(timeoutConns))
	}
	seen := make(map[string]bool)
	for _, c := range timeoutConns {
		seen[c] = true
	}
	if !seen["c_old"] {
		t.Error("c_old should be in timeout list")
	}
	if !seen["c_older"] {
		t.Error("c_older should be in timeout list")
	}
	if seen["c_fresh"] {
		t.Error("c_fresh should NOT be in timeout list")
	}
}

func TestHeartbeat_CheckTimeouts_AtomicLastHeartbeat(t *testing.T) {
	mgr := NewManager()
	conn := &Connection{
		ConnID: "c1",
		UserID: "u1",
	}
	atomic.StoreInt64(&conn.LastHeartbeat, time.Now().UnixMilli())
	mgr.Add(context.Background(), conn)

	cfg := HeartbeatConfig{Interval: time.Second, Timeout: time.Minute}
	hb := NewHeartbeat(mgr, cfg)

	// Update heartbeat atomically
	atomic.StoreInt64(&conn.LastHeartbeat, time.Now().UnixMilli())

	timeoutCalled := false
	hb.checkTimeouts(context.Background(), func(_ context.Context, connID string) {
		timeoutCalled = true
	})
	if timeoutCalled {
		t.Error("onTimeout should not be called when heartbeat was updated atomically")
	}
}

func TestHeartbeat_CheckTimeouts_NilOnTimeout(t *testing.T) {
	mgr := NewManager()
	mgr.Add(context.Background(), &Connection{
		ConnID:        "c1",
		UserID:        "u1",
		LastHeartbeat: time.Now().Add(-2 * time.Minute).UnixMilli(),
	})

	cfg := HeartbeatConfig{Interval: time.Second, Timeout: 30 * time.Second}
	hb := NewHeartbeat(mgr, cfg)

	// Should not panic when onTimeout is nil
	hb.checkTimeouts(context.Background(), nil)

	if !mgr.Get(context.Background(), "c1").IsClosed() {
		t.Error("connection should be closed after timeout check")
	}
}
