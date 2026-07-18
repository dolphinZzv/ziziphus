package gateway

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"ziziphus/internal/metrics"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/protocol"
)

type Manager struct {
	mu        sync.RWMutex
	conns     map[string]*Connection            // connID -> Connection
	userConns map[string]map[string]*Connection // userID -> connID -> Connection
	sessConns map[string]*Connection            // sessionID -> Connection
}

func NewManager() *Manager {
	return &Manager{
		conns:     make(map[string]*Connection),
		userConns: make(map[string]map[string]*Connection),
		sessConns: make(map[string]*Connection),
	}
}

func (m *Manager) Add(_ context.Context, conn *Connection) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.conns[conn.ConnID] = conn
	if m.userConns[conn.UserID] == nil {
		m.userConns[conn.UserID] = make(map[string]*Connection)
	}
	m.userConns[conn.UserID][conn.ConnID] = conn
	m.sessConns[conn.SessionID] = conn
	metrics.ConnectionsTotal.Inc()
	logger.Info("connection added", "conn_id", conn.ConnID, "user_id", conn.UserID)
}

func (m *Manager) Remove(_ context.Context, connID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	conn, ok := m.conns[connID]
	if !ok {
		return
	}
	delete(m.conns, connID)
	if userMap, ok := m.userConns[conn.UserID]; ok {
		delete(userMap, connID)
		if len(userMap) == 0 {
			delete(m.userConns, conn.UserID)
		}
	}
	delete(m.sessConns, conn.SessionID)
	metrics.ConnectionsTotal.Dec()
	logger.Info("connection removed", "conn_id", connID)
}

func (m *Manager) Get(_ context.Context, connID string) *Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.conns[connID]
}

func (m *Manager) GetBySessionID(_ context.Context, sessionID string) any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn := m.sessConns[sessionID]
	if conn == nil {
		return nil
	}
	return conn
}

// DisconnectBySessionID force-disconnects a WebSocket connection by session ID
// and removes it from the manager. Used for session revocation.
func (m *Manager) DisconnectBySessionID(ctx context.Context, sessionID string) {
	m.mu.Lock()
	conn, ok := m.sessConns[sessionID]
	if !ok {
		m.mu.Unlock()
		return
	}
	connID := conn.ConnID
	userID := conn.UserID
	// Remove from sessConns under lock, but defer Close()/Sleep() until after unlock
	delete(m.sessConns, sessionID)
	m.mu.Unlock()

	// Send kick frame and close outside the lock to avoid blocking other operations
	errPayload, _ := json.Marshal(protocol.ErrorPayload{Code: 4001, Message: "kicked"})
	_ = conn.SendFrame(protocol.Frame{Type: protocol.Error, Payload: errPayload})
	time.Sleep(100 * time.Millisecond)
	conn.Close()

	// Re-acquire lock to clean up remaining maps
	m.mu.Lock()
	delete(m.conns, connID)
	if userMap, ok2 := m.userConns[userID]; ok2 {
		delete(userMap, connID)
		if len(userMap) == 0 {
			delete(m.userConns, userID)
		}
	}
	m.mu.Unlock()
	metrics.ConnectionsTotal.Dec()
	logger.Info("connection force-disconnected", "session_id", sessionID, "conn_id", connID)
}

func (m *Manager) GetByUserID(_ context.Context, userID string) []any {
	m.mu.RLock()
	defer m.mu.RUnlock()
	userMap, ok := m.userConns[userID]
	if !ok {
		return nil
	}
	conns := make([]any, 0, len(userMap))
	for _, conn := range userMap {
		conns = append(conns, conn)
	}
	return conns
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.conns)
}

func (m *Manager) All() []*Connection {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conns := make([]*Connection, 0, len(m.conns))
	for _, conn := range m.conns {
		conns = append(conns, conn)
	}
	return conns
}
