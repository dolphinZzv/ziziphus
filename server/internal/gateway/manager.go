package gateway

import (
	"context"
	"sync"

	"github.com/dolphinz/im-server/internal/metrics"
	"github.com/dolphinz/im-server/pkg/logger"
)

type Manager struct {
	mu        sync.RWMutex
	conns     map[string]*Connection          // connID -> Connection
	userConns map[string]map[string]*Connection // userID -> connID -> Connection
	sessConns map[string]*Connection          // sessionID -> Connection
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
