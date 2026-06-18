package session

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type Manager struct {
	mu       sync.RWMutex
	registry map[string]*model.Session      // sessionID -> Session
	userSess map[string]map[string]struct{} // userID -> set of sessionIDs

	sessionCache sessionCache
	sessionRepo  sessionRepo
}

type sessionCache interface {
	Set(ctx context.Context, s *model.Session) error
	Get(ctx context.Context, sessionID string) (*model.Session, error)
	Delete(ctx context.Context, sessionID, userID string) error
	GetUserSessionIDs(ctx context.Context, userID string) ([]string, error)
}

type sessionRepo interface {
	Create(ctx context.Context, s *model.Session) error
	Delete(ctx context.Context, sessionID string) error
}

func NewManager(sessionCache sessionCache, sessionRepo sessionRepo) *Manager {
	return &Manager{
		registry:     make(map[string]*model.Session),
		userSess:     make(map[string]map[string]struct{}),
		sessionCache: sessionCache,
		sessionRepo:  sessionRepo,
	}
}

func (m *Manager) Create(ctx context.Context, userID string, device model.DeviceType, deviceName string, clientIP string, deviceID string) (*model.Session, error) {
	sessionID := "sess_" + uuid.New().String()[:8]
	s := &model.Session{
		SessionID:  sessionID,
		UserID:     userID,
		Device:     device,
		DeviceName: deviceName,
		DeviceID:   deviceID,
		ClientIP:   clientIP,
		Status:     model.SessionActive,
		LoginAt:    time.Now().UnixMilli(),
		LastActive: time.Now().UnixMilli(),
	}

	if err := m.sessionRepo.Create(ctx, s); err != nil {
		return nil, err
	}
	if err := m.sessionCache.Set(ctx, s); err != nil {
		logger.Warn("session cache set failed", "session_id", sessionID, "error", err)
	}

	m.mu.Lock()
	m.registry[sessionID] = s
	if m.userSess[userID] == nil {
		m.userSess[userID] = make(map[string]struct{})
	}
	m.userSess[userID][sessionID] = struct{}{}
	m.mu.Unlock()

	logger.Info("session created", "session_id", sessionID, "user_id", userID, "device", device)
	return s, nil
}

func (m *Manager) Get(ctx context.Context, sessionID string) *model.Session {
	m.mu.RLock()
	s, ok := m.registry[sessionID]
	m.mu.RUnlock()
	if ok {
		return s
	}
	// fallback to cache
	cached, err := m.sessionCache.Get(ctx, sessionID)
	if err != nil {
		return nil
	}
	m.mu.Lock()
	m.registry[sessionID] = cached
	m.mu.Unlock()
	return cached
}

func (m *Manager) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	s, ok := m.registry[sessionID]
	if ok {
		delete(m.userSess[s.UserID], sessionID)
		if len(m.userSess[s.UserID]) == 0 {
			delete(m.userSess, s.UserID)
		}
		delete(m.registry, sessionID)
	}
	m.mu.Unlock()
	if ok {
		if err := m.sessionCache.Delete(ctx, sessionID, s.UserID); err != nil {
			logger.Warn("session cache delete failed", "session_id", sessionID, "error", err)
		}
		if err := m.sessionRepo.Delete(ctx, sessionID); err != nil {
			logger.Error("session repo delete failed", "session_id", sessionID, "error", err)
		}
	}
	return nil
}

func (m *Manager) BindConnection(ctx context.Context, sessionID, connID string) error {
	m.mu.Lock()
	s, ok := m.registry[sessionID]
	if ok {
		s.ConnID = connID
		s.LastActive = time.Now().UnixMilli()
	}
	m.mu.Unlock()
	if !ok {
		return model.NewAppError(model.ErrNotFound, "session not found")
	}
	// update cache
	if cached, err := m.sessionCache.Get(ctx, sessionID); err == nil {
		cached.ConnID = connID
		m.sessionCache.Set(ctx, cached)
	}
	return nil
}

func (m *Manager) IsOnline(ctx context.Context, userID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessions, ok := m.userSess[userID]
	if !ok || len(sessions) == 0 {
		return false
	}
	for sid := range sessions {
		s := m.registry[sid]
		if s != nil && s.Status == model.SessionActive {
			return true
		}
	}
	return false
}

func (m *Manager) GetUserSessionIDs(ctx context.Context, userID string) []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	sessions, ok := m.userSess[userID]
	if !ok {
		return nil
	}
	ids := make([]string, 0, len(sessions))
	for sid := range sessions {
		ids = append(ids, sid)
	}
	return ids
}
