package session

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"ziziphus/pkg/model"
)

// ---------------------------------------------------------------------------
// Mock implementations
// ---------------------------------------------------------------------------

type mockCache struct {
	mu           sync.Mutex
	sessions     map[string]*model.Session
	userSess     map[string]map[string]struct{} // userID -> set of sessionIDs
	setErrVal    error                          // injected error for Set (persistent)
	deleteErrVal error                          // injected error for Delete (persistent)
}

func newMockCache() *mockCache {
	return &mockCache{
		sessions: make(map[string]*model.Session),
		userSess: make(map[string]map[string]struct{}),
	}
}

// setErr sets a persistent error that will be returned by all subsequent Set calls.
func (c *mockCache) setErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.setErrVal = err
}

func (c *mockCache) setDeleteErr(err error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deleteErrVal = err
}

func (c *mockCache) Set(_ context.Context, s *model.Session) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.setErrVal != nil {
		return c.setErrVal
	}
	c.sessions[s.SessionID] = s
	if c.userSess[s.UserID] == nil {
		c.userSess[s.UserID] = make(map[string]struct{})
	}
	c.userSess[s.UserID][s.SessionID] = struct{}{}
	return nil
}

func (c *mockCache) Get(_ context.Context, sessionID string) (*model.Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	s, ok := c.sessions[sessionID]
	if !ok {
		return nil, fmt.Errorf("mockCache: session %s not found", sessionID)
	}
	return s, nil
}

func (c *mockCache) Delete(_ context.Context, sessionID, userID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.deleteErrVal != nil {
		return c.deleteErrVal
	}
	delete(c.sessions, sessionID)
	if c.userSess[userID] != nil {
		delete(c.userSess[userID], sessionID)
		if len(c.userSess[userID]) == 0 {
			delete(c.userSess, userID)
		}
	}
	return nil
}

func (c *mockCache) GetUserSessionIDs(_ context.Context, userID string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids, ok := c.userSess[userID]
	if !ok {
		return nil, nil
	}
	result := make([]string, 0, len(ids))
	for id := range ids {
		result = append(result, id)
	}
	return result, nil
}

// len returns the number of sessions in the cache.
func (c *mockCache) len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.sessions)
}

// hasSession returns true if the cache contains the given sessionID.
func (c *mockCache) hasSession(sessionID string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, ok := c.sessions[sessionID]
	return ok
}

// ---------------------------------------------------------------------------

type mockRepo struct {
	mu        sync.Mutex
	sessions  map[string]*model.Session
	createErr error // injected error for Create
	deleteErr error // injected error for Delete
}

func newMockRepo() *mockRepo {
	return &mockRepo{
		sessions: make(map[string]*model.Session),
	}
}

func (r *mockRepo) Create(_ context.Context, s *model.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.createErr != nil {
		return r.createErr
	}
	r.sessions[s.SessionID] = s
	return nil
}

func (r *mockRepo) Delete(_ context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.deleteErr != nil {
		return r.deleteErr
	}
	delete(r.sessions, sessionID)
	return nil
}

func (r *mockRepo) len() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.sessions)
}

func (r *mockRepo) hasSession(sessionID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	_, ok := r.sessions[sessionID]
	return ok
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

var ctx = context.Background()

// newManager creates a Manager backed by fresh mockCache and mockRepo.
func newManager() (*Manager, *mockCache, *mockRepo) {
	c := newMockCache()
	r := newMockRepo()
	return NewManager(c, r), c, r
}

// ---------------------------------------------------------------------------
// Tests: NewManager
// ---------------------------------------------------------------------------

func TestNewManager(t *testing.T) {
	m, c, r := newManager()
	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.registry == nil {
		t.Error("registry map is nil")
	}
	if m.userSess == nil {
		t.Error("userSess map is nil")
	}
	if m.sessionCache != c {
		t.Error("sessionCache not set correctly")
	}
	if m.sessionRepo != r {
		t.Error("sessionRepo not set correctly")
	}
}

// ---------------------------------------------------------------------------
// Tests: Create
// ---------------------------------------------------------------------------

func TestCreate_Success(t *testing.T) {
	m, cache, repo := newManager()

	s, err := m.Create(ctx, "user1", model.DevicePhone, "iPhone 15", "", "")
	if err != nil {
		t.Fatalf("Create returned unexpected error: %v", err)
	}

	// Verify returned session fields
	if s == nil {
		t.Fatal("Create returned nil session")
	}
	if !strings.HasPrefix(s.SessionID, "sess_") || len(s.SessionID) != 13 {
		t.Errorf("unexpected session ID format: %q", s.SessionID)
	}
	if s.UserID != "user1" {
		t.Errorf("UserID = %q, want %q", s.UserID, "user1")
	}
	if s.Device != model.DevicePhone {
		t.Errorf("Device = %v, want %v", s.Device, model.DevicePhone)
	}
	if s.DeviceName != "iPhone 15" {
		t.Errorf("DeviceName = %q, want %q", s.DeviceName, "iPhone 15")
	}
	if s.Status != model.SessionActive {
		t.Errorf("Status = %v, want %v", s.Status, model.SessionActive)
	}
	if s.LoginAt == 0 {
		t.Error("LoginAt should be set")
	}
	if s.LastActive == 0 {
		t.Error("LastActive should be set")
	}

	// Verify stored in local registry
	m.mu.RLock()
	regS, inReg := m.registry[s.SessionID]
	userMap, userExists := m.userSess["user1"]
	m.mu.RUnlock()
	if !inReg {
		t.Error("session not found in local registry")
	} else if regS != s {
		t.Error("registry holds a different pointer")
	}
	if !userExists {
		t.Error("user1 not in userSess map")
	} else if _, ok := userMap[s.SessionID]; !ok {
		t.Error("session ID not in userSess[user1]")
	}

	// Verify stored in cache
	if !cache.hasSession(s.SessionID) {
		t.Error("session not found in mock cache")
	}

	// Verify stored in repo
	if !repo.hasSession(s.SessionID) {
		t.Error("session not found in mock repo")
	}
}

func TestCreate_RepoError(t *testing.T) {
	m, cache, repo := newManager()

	// Inject a repo error
	repo.createErr = fmt.Errorf("database unavailable")

	s, err := m.Create(ctx, "user2", model.DeviceDesktop, "MacBook", "", "")
	if err == nil {
		t.Fatal("expected error but got nil")
	}
	if s != nil {
		t.Fatal("expected nil session on error")
	}

	// Verify nothing was stored locally
	m.mu.RLock()
	regLen := len(m.registry)
	userLen := len(m.userSess)
	m.mu.RUnlock()
	if regLen != 0 {
		t.Errorf("expected empty registry, got %d entries", regLen)
	}
	if userLen != 0 {
		t.Errorf("expected empty userSess, got %d entries", userLen)
	}

	// Verify nothing was stored in cache
	if cache.len() != 0 {
		t.Error("expected empty cache on repo error")
	}

	// Verify nothing was stored in repo
	if repo.len() != 0 {
		t.Error("expected empty repo (create itself failed)")
	}
}

// ---------------------------------------------------------------------------
// Tests: Create (continued) — error-log-only paths
// ---------------------------------------------------------------------------

func TestCreate_CacheSetError(t *testing.T) {
	// When cache.Set fails, Create should still succeed because the error is
	// only logged, not returned.
	m, cache, repo := newManager()

	// Make cache.Set fail
	cache.setErr(fmt.Errorf("redis timeout"))

	s, err := m.Create(ctx, "user_cachefail", model.DeviceWeb, "Firefox", "", "")
	if err != nil {
		t.Fatalf("Create should succeed even when cache.Set fails: %v", err)
	}
	if s == nil {
		t.Fatal("Create returned nil session")
	}

	// Session should still be in registry and repo
	m.mu.RLock()
	_, inReg := m.registry[s.SessionID]
	m.mu.RUnlock()
	if !inReg {
		t.Error("session not in registry despite cache.Set failure")
	}
	if !repo.hasSession(s.SessionID) {
		t.Error("session not in repo despite cache.Set failure")
	}

	// Session should NOT be in cache (Set failed)
	if cache.hasSession(s.SessionID) {
		t.Error("session unexpectedly found in cache after Set failure")
	}
}

func TestGet_FromRegistry(t *testing.T) {
	m, cache, repo := newManagerWithSession(t, "user3", model.DeviceWeb, "Chrome")

	// Retrieve the existing session to learn its ID
	m.mu.RLock()
	var sessID string
	for id := range m.registry {
		sessID = id
		break
	}
	m.mu.RUnlock()

	// Remove from cache so the fallback path would fail if called
	cache.Delete(ctx, sessID, "user3")
	repo.Delete(ctx, sessID)

	got := m.Get(ctx, sessID)
	if got == nil {
		t.Fatal("Get returned nil for session in registry")
	}
	if got.SessionID != sessID {
		t.Errorf("Get returned session with ID %q, want %q", got.SessionID, sessID)
	}
}

func TestGet_FallbackToCache(t *testing.T) {
	m, cache, _ := newManager()

	// Pre-populate the cache directly (session is NOT in registry)
	s := &model.Session{
		SessionID:  "sess_cacheonly",
		UserID:     "user4",
		Device:     model.DevicePhone,
		DeviceName: "Pixel",
		Status:     model.SessionActive,
		LoginAt:    time.Now().UnixMilli(),
		LastActive: time.Now().UnixMilli(),
	}
	cache.Set(ctx, s)

	got := m.Get(ctx, "sess_cacheonly")
	if got == nil {
		t.Fatal("Get returned nil, expected session from cache")
	}
	if got.SessionID != "sess_cacheonly" {
		t.Errorf("Get returned session with ID %q, want %q", got.SessionID, "sess_cacheonly")
	}

	// Verify it was promoted to local registry
	m.mu.RLock()
	regS, inReg := m.registry["sess_cacheonly"]
	m.mu.RUnlock()
	if !inReg {
		t.Error("session was not promoted to local registry after cache hit")
	} else if regS != got {
		t.Error("registry pointer differs from returned pointer")
	}
}

func TestGet_NotFound(t *testing.T) {
	m, _, _ := newManager()

	got := m.Get(ctx, "sess_nonexistent")
	if got != nil {
		t.Fatalf("Get returned %v, want nil", got)
	}
}

// ---------------------------------------------------------------------------
// Tests: Delete
// ---------------------------------------------------------------------------

func TestDelete_Existing(t *testing.T) {
	m, cache, repo := newManagerWithSession(t, "user5", model.DeviceDesktop, "Windows PC")

	m.mu.RLock()
	var sessID, userID string
	for id, s := range m.registry {
		sessID = id
		userID = s.UserID
		break
	}
	m.mu.RUnlock()

	err := m.Delete(ctx, sessID)
	if err != nil {
		t.Fatalf("Delete returned unexpected error: %v", err)
	}

	// Verify removed from registry
	m.mu.RLock()
	_, inReg := m.registry[sessID]
	_, userExists := m.userSess[userID]
	m.mu.RUnlock()
	if inReg {
		t.Error("session still in registry after Delete")
	}
	if userExists {
		t.Error("userSess entry was not cleaned up after Delete")
	}

	// Verify removed from cache
	if cache.hasSession(sessID) {
		t.Error("session still in cache after Delete")
	}

	// Verify removed from repo
	if repo.hasSession(sessID) {
		t.Error("session still in repo after Delete")
	}
}

func TestDelete_NonExistent(t *testing.T) {
	m, cache, repo := newManager()

	// Should not panic and should return nil
	err := m.Delete(ctx, "sess_doesnotexist")
	if err != nil {
		t.Fatalf("Delete returned error for non-existent session: %v", err)
	}

	if cache.len() != 0 {
		t.Error("cache should be empty")
	}
	if repo.len() != 0 {
		t.Error("repo should be empty")
	}
}

// ---------------------------------------------------------------------------
// Tests: Delete (continued) — error-log-only paths
// ---------------------------------------------------------------------------

func TestDelete_CacheDeleteError(t *testing.T) {
	m, cache, repo := newManagerWithSession(t, "del_cachefail", model.DevicePhone, "Nexus")

	m.mu.RLock()
	var sessID, userID string
	for id, s := range m.registry {
		sessID = id
		userID = s.UserID
		break
	}
	m.mu.RUnlock()

	// Make cache.Delete fail
	cache.setDeleteErr(fmt.Errorf("cache unavailable"))

	err := m.Delete(ctx, sessID)
	if err != nil {
		t.Fatalf("Delete should return nil even when cache.Delete fails: %v", err)
	}

	// Local state should still be cleaned up
	m.mu.RLock()
	_, inReg := m.registry[sessID]
	_, userExists := m.userSess[userID]
	m.mu.RUnlock()
	if inReg {
		t.Error("session still in registry despite cache.Delete failure")
	}
	if userExists {
		t.Error("userSess entry was not cleaned up despite cache.Delete failure")
	}

	// Repo should still have been called and removed the session
	if repo.hasSession(sessID) {
		t.Error("session still in repo despite cache.Delete failure")
	}
}

func TestDelete_RepoDeleteError(t *testing.T) {
	m, cache, repo := newManagerWithSession(t, "del_repofail", model.DeviceDesktop, "ThinkPad")

	m.mu.RLock()
	var sessID, userID string
	for id, s := range m.registry {
		sessID = id
		userID = s.UserID
		break
	}
	m.mu.RUnlock()

	// Make repo.Delete fail
	repo.deleteErr = fmt.Errorf("db connection lost")

	err := m.Delete(ctx, sessID)
	if err != nil {
		t.Fatalf("Delete should return nil even when repo.Delete fails: %v", err)
	}

	// Local state and cache should still be cleaned up
	m.mu.RLock()
	_, inReg := m.registry[sessID]
	_, userExists := m.userSess[userID]
	m.mu.RUnlock()
	if inReg {
		t.Error("session still in registry despite repo.Delete failure")
	}
	if userExists {
		t.Error("userSess entry was not cleaned up despite repo.Delete failure")
	}

	// Cache should have been cleaned up before the repo error
	if cache.hasSession(sessID) {
		t.Error("session still in cache despite repo.Delete failure")
	}
}

// ---------------------------------------------------------------------------
// Tests: BindConnection
// ---------------------------------------------------------------------------

func TestBindConnection_Success(t *testing.T) {
	m, cache, _ := newManagerWithSession(t, "user6", model.DevicePhone, "Galaxy")

	m.mu.RLock()
	var sessID string
	for id := range m.registry {
		sessID = id
		break
	}
	m.mu.RUnlock()

	// Record the LastActive before binding
	before := time.Now().UnixMilli()
	time.Sleep(2 * time.Millisecond) // ensure time advances

	err := m.BindConnection(ctx, sessID, "conn_abc123")
	if err != nil {
		t.Fatalf("BindConnection returned error: %v", err)
	}

	// Verify registry was updated
	m.mu.RLock()
	s := m.registry[sessID]
	m.mu.RUnlock()
	if s == nil {
		t.Fatal("session missing from registry after BindConnection")
	}
	if s.ConnID != "conn_abc123" {
		t.Errorf("ConnID = %q, want %q", s.ConnID, "conn_abc123")
	}
	if s.LastActive <= before {
		t.Error("LastActive was not updated after BindConnection")
	}

	// Verify cache was also updated
	cached, err := cache.Get(ctx, sessID)
	if err != nil {
		t.Fatalf("session missing from cache after BindConnection: %v", err)
	}
	if cached.ConnID != "conn_abc123" {
		t.Errorf("cache ConnID = %q, want %q", cached.ConnID, "conn_abc123")
	}
}

func TestBindConnection_NotFound(t *testing.T) {
	m, _, _ := newManager()

	err := m.BindConnection(ctx, "sess_unknown", "conn_xyz")
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}

	// Verify it's an AppError with ErrNotFound
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNotFound {
		t.Errorf("error code = %d, want %d", appErr.Code, model.ErrNotFound)
	}
}

// ---------------------------------------------------------------------------
// Tests: IsOnline
// ---------------------------------------------------------------------------

func TestIsOnline_ActiveSession(t *testing.T) {
	m, _, _ := newManagerWithSession(t, "online_user", model.DeviceDesktop, "iMac")

	if !m.IsOnline(ctx, "online_user") {
		t.Error("IsOnline returned false, expected true for user with active session")
	}
}

func TestIsOnline_NoSessions(t *testing.T) {
	m, _, _ := newManager()

	if m.IsOnline(ctx, "ghost_user") {
		t.Error("IsOnline returned true for user with no sessions")
	}
}

func TestIsOnline_InactiveSessions(t *testing.T) {
	m, _, _ := newManager()

	// Manually inject a session with Inactive status into internal maps
	sessID := "sess_inactive"
	s := &model.Session{
		SessionID:  sessID,
		UserID:     "lazy_user",
		Device:     model.DeviceWeb,
		DeviceName: "Safari",
		Status:     model.SessionInactive,
		LoginAt:    time.Now().UnixMilli(),
		LastActive: time.Now().UnixMilli(),
	}
	m.mu.Lock()
	m.registry[sessID] = s
	m.userSess["lazy_user"] = map[string]struct{}{sessID: {}}
	m.mu.Unlock()

	if m.IsOnline(ctx, "lazy_user") {
		t.Error("IsOnline returned true for user with only inactive sessions")
	}
}

// ---------------------------------------------------------------------------
// Tests: GetUserSessionIDs
// ---------------------------------------------------------------------------

func TestGetUserSessionIDs(t *testing.T) {
	m, _, _ := newManager()

	// Create two sessions for the same user
	s1, err := m.Create(ctx, "multi_user", model.DevicePhone, "iPhone", "", "")
	if err != nil {
		t.Fatalf("Create session 1 failed: %v", err)
	}
	s2, err := m.Create(ctx, "multi_user", model.DeviceDesktop, "MacBook", "", "")
	if err != nil {
		t.Fatalf("Create session 2 failed: %v", err)
	}

	ids := m.GetUserSessionIDs(ctx, "multi_user")
	if ids == nil {
		t.Fatal("GetUserSessionIDs returned nil")
	}
	if len(ids) != 2 {
		t.Fatalf("expected 2 session IDs, got %d: %v", len(ids), ids)
	}

	// The slice should contain both session IDs (order is non-deterministic due to map iteration)
	idMap := make(map[string]bool, len(ids))
	for _, id := range ids {
		idMap[id] = true
	}
	if !idMap[s1.SessionID] {
		t.Errorf("missing session ID %q in result", s1.SessionID)
	}
	if !idMap[s2.SessionID] {
		t.Errorf("missing session ID %q in result", s2.SessionID)
	}
}

func TestGetUserSessionIDs_UnknownUser(t *testing.T) {
	m, _, _ := newManager()

	ids := m.GetUserSessionIDs(ctx, "nobody")
	if ids != nil {
		t.Fatalf("expected nil for unknown user, got %v", ids)
	}
}

// ---------------------------------------------------------------------------
// Tests: Delete with multiple sessions per user (edge case)
// ---------------------------------------------------------------------------

func TestDelete_OnlyOneSessionOfMultiple(t *testing.T) {
	m, cache, repo := newManager()

	s1, _ := m.Create(ctx, "multi_user2", model.DevicePhone, "Phone", "", "")
	s2, _ := m.Create(ctx, "multi_user2", model.DeviceDesktop, "Desktop", "", "")

	// Delete only the first session
	err := m.Delete(ctx, s1.SessionID)
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify the remaining session still exists in registry
	m.mu.RLock()
	_, s1InReg := m.registry[s1.SessionID]
	_, s2InReg := m.registry[s2.SessionID]
	userSess, userExists := m.userSess["multi_user2"]
	m.mu.RUnlock()
	if s1InReg {
		t.Error("deleted session still in registry")
	}
	if !s2InReg {
		t.Error("remaining session missing from registry")
	}
	if !userExists {
		t.Error("user entry should still exist (other session remains)")
	} else if len(userSess) != 1 {
		t.Errorf("expected 1 session in userSess, got %d", len(userSess))
	}

	// Verify cache
	if cache.hasSession(s1.SessionID) {
		t.Error("deleted session still in cache")
	}
	if !cache.hasSession(s2.SessionID) {
		t.Error("remaining session missing from cache")
	}

	// Verify repo
	if repo.hasSession(s1.SessionID) {
		t.Error("deleted session still in repo")
	}
	if !repo.hasSession(s2.SessionID) {
		t.Error("remaining session missing from repo")
	}
}

// ---------------------------------------------------------------------------
// Helper: create a session and return the Manager + mocks
// ---------------------------------------------------------------------------

func newManagerWithSession(t *testing.T, userID string, device model.DeviceType, deviceName string) (*Manager, *mockCache, *mockRepo) {
	t.Helper()
	m, c, r := newManager()
	s, err := m.Create(ctx, userID, device, deviceName, "", "")
	if err != nil {
		t.Fatalf("helper Create failed: %v", err)
	}
	if s == nil {
		t.Fatal("helper Create returned nil session")
	}
	return m, c, r
}
