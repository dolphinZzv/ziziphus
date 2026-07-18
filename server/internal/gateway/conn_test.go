package gateway

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
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
	// CreatedAt and LastHeartbeat should be set to the current time
	c := NewConnection("c1", "u1", "s1", 1, nil)
	if c.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if c.LastHeartbeat == 0 {
		t.Error("LastHeartbeat should be set")
	}
}

func TestConnection_SendJSON_NilReceiver(t *testing.T) {
	var c *Connection
	err := c.SendJSON(map[string]string{"key": "val"})
	if err != nil {
		t.Errorf("SendJSON on nil receiver: %v", err)
	}
}

func TestConnection_SendJSON_NilPayload(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	err := c.SendJSON(nil)
	if err != nil {
		t.Errorf("SendJSON with nil payload: %v", err)
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
	if err := c.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := c.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestConnection_IsClosed_AfterClose(t *testing.T) {
	c := NewConnection("c1", "u1", "s1", 1, nil)
	_ = c.Close()
	if !c.IsClosed() {
		t.Error("connection should be closed after Close()")
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

// NewConnection calls SetReadLimit(1MB). We verify it works
// by creating a real WebSocket pair and confirming the connection
// is usable after NewConnection.
func TestNewConnection_SetsReadLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{}
		wsConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		// Server side — NewConnection applies SetReadLimit(1MB)
		c := NewConnection("srv", "u1", "s1", 1, wsConn)
		defer c.Close()
		_, _, _ = wsConn.ReadMessage()
	}))
	defer srv.Close()

	wsURL := "ws" + srv.URL[4:] + "/ws"
	dialer := websocket.Dialer{}
	wsConn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer wsConn.Close()

	// A small message should work fine
	err = wsConn.WriteMessage(websocket.TextMessage, []byte("hello"))
	if err != nil {
		t.Errorf("small message should succeed: %v", err)
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
