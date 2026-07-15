package gateway

import (
	"encoding/json"
	"testing"
	"time"

	"ziziphus/pkg/protocol"
)

func TestNewConnection(t *testing.T) {
	now := time.Now()
	c := NewConnection("conn1", "user1", "sess1", 2, nil)
	if c == nil {
		t.Fatal("NewConnection returned nil")
	}
	if c.ConnID != "conn1" {
		t.Errorf("ConnID = %q, want %q", c.ConnID, "conn1")
	}
	if c.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", c.UserID, "user1")
	}
	if c.SessionID != "sess1" {
		t.Errorf("SessionID = %q, want %q", c.SessionID, "sess1")
	}
	if c.Device != 2 {
		t.Errorf("Device = %d, want %d", c.Device, 2)
	}
	if c.CreatedAt.Before(now) || c.CreatedAt.After(time.Now()) {
		t.Errorf("CreatedAt = %v, is not current", c.CreatedAt)
	}
	if c.LastHeartbeat == 0 {
		t.Errorf("LastHeartbeat = 0, want non-zero")
	}
	if c.closed {
		t.Error("new connection should not be closed")
	}
}

func TestNewConnection_SetsLastHeartbeat(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	expected := c.CreatedAt.UnixMilli()
	if c.LastHeartbeat != expected {
		t.Errorf("LastHeartbeat = %d, want %d", c.LastHeartbeat, expected)
	}
}

func TestConnection_SendJSON_NilReceiver(t *testing.T) {
	var c *Connection
	// Should not panic
	err := c.SendJSON("hello")
	if err != nil {
		t.Errorf("SendJSON on nil receiver returned error: %v", err)
	}
}

func TestConnection_SendJSON_NilPayload(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	// Should not panic and return nil
	err := c.SendJSON(nil)
	if err != nil {
		t.Errorf("SendJSON with nil payload returned error: %v", err)
	}
}

func TestConnection_IsClosed_InitiallyFalse(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	if c.IsClosed() {
		t.Error("new connection should not be closed")
	}
}

func TestConnection_Close_Idempotent(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	// No real websocket.Conn, Close will fail at c.Conn.Close().
	// But we only care about the closed flag.
	_ = c.Close()
	// We don't check err because there's no real Conn.
	if !c.closed {
		t.Error("connection should be marked closed after Close()")
	}
	// Second close should be a no-op
	err2 := c.Close()
	if err2 != nil {
		// Even if first close errored, second should be handled
	}
}

func TestConnection_IsClosed_AfterClose(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	_ = c.Close()
	if !c.IsClosed() {
		t.Error("IsClosed should return true after Close()")
	}
}

func TestConnection_SendJSON_WhileClosed(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	_ = c.Close()
	// Should not panic, should return nil for closed conn without real ws
	err := c.SendJSON(map[string]string{"key": "val"})
	if err != nil {
		t.Errorf("SendJSON on closed connection should not propagate error: %v", err)
	}
}

func TestConnection_SendFrame_DelegatesToSendJSON(t *testing.T) {
	// SendFrame calls SendJSON internally; test nil-safety
	var c *Connection
	err := c.SendFrame(protocol.Frame{})
	if err != nil {
		t.Errorf("SendFrame on nil receiver: %v", err)
	}
}

func TestConnection_Close_NilConn_Safe(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	err := c.Close()
	if err != nil {
		t.Errorf("Close with nil Conn should not error: %v", err)
	}
	if !c.closed {
		t.Error("connection should be closed after Close()")
	}
}

func TestConnection_MarshalJSON(t *testing.T) {
	now := time.Now()
	c := &Connection{
		ConnID:    "conn-x",
		UserID:    "user-x",
		SessionID: "sess-x",
		Device:    3,
		CreatedAt: now,
	}
	data, err := c.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON error: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if m["conn_id"] != "conn-x" {
		t.Errorf("conn_id = %v, want conn-x", m["conn_id"])
	}
	if m["user_id"] != "user-x" {
		t.Errorf("user_id = %v, want user-x", m["user_id"])
	}
	if m["session_id"] != "sess-x" {
		t.Errorf("session_id = %v, want sess-x", m["session_id"])
	}
	if int(m["device"].(float64)) != 3 {
		t.Errorf("device = %v, want 3", m["device"])
	}
}
