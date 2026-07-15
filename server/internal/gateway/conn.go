package gateway

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/protocol"
)

type Connection struct {
	mu            sync.Mutex
	ConnID        string
	UserID        string
	SessionID     string
	Device        int
	Conn          *websocket.Conn
	CreatedAt     time.Time
	LastHeartbeat int64
	closed        bool
}

func NewConnection(connID, userID, sessionID string, device int, conn *websocket.Conn) *Connection {
	now := time.Now()
	return &Connection{
		ConnID:        connID,
		UserID:        userID,
		SessionID:     sessionID,
		Device:        device,
		Conn:          conn,
		CreatedAt:     now,
		LastHeartbeat: now.UnixMilli(),
	}
}

func (c *Connection) SendJSON(v interface{}) error {
	if c == nil {
		return nil
	}
	if v == nil {
		logger.Warn("SendJSON called with nil payload", "conn_id", c.ConnID, "user_id", c.UserID)
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	return c.Conn.WriteJSON(v)
}

func (c *Connection) SendFrame(frame protocol.Frame) error {
	return c.SendJSON(frame)
}

func (c *Connection) ReadFrame() (protocol.Frame, error) {
	var frame protocol.Frame
	err := c.Conn.ReadJSON(&frame)
	return frame, err
}

func (c *Connection) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return nil
	}
	c.closed = true
	if c.Conn == nil {
		return nil
	}
	return c.Conn.Close()
}

func (c *Connection) IsClosed() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.closed
}

func (c *Connection) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"conn_id":    c.ConnID,
		"user_id":    c.UserID,
		"session_id": c.SessionID,
		"device":     c.Device,
		"created_at": c.CreatedAt,
	})
}
