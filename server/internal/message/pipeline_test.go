package message

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	pgx "github.com/jackc/pgx/v5"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

// ---------------------------------------------------------------------------
// Mock types
// ---------------------------------------------------------------------------

// mockConn implements interface{ SendFrame(protocol.Frame) error } used in
// pusher and receipt-handler type assertions.
type mockConn struct {
	mu      sync.Mutex
	frames  []protocol.Frame
	sendErr error
}

func (c *mockConn) SendFrame(frame protocol.Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.frames = append(c.frames, frame)
	return c.sendErr
}

func (c *mockConn) getFrames() []protocol.Frame {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]protocol.Frame, len(c.frames))
	copy(out, c.frames)
	return out
}

// mockMessageStore implements messageStore, receiptMsgRepo, and syncMsgRepo.
type mockMessageStore struct {
	mu         sync.Mutex
	messages   map[int64]*model.Message // msgID -> msg
	clientSeqs map[string]*model.Message
	allMsgs    []*model.Message // ordered insertion for sync queries

	insertErr error
	getErr    error
	syncErr   error

	insertCalls int
	getCalls    int
	syncCalls   int
}

// messageStore methods

func (m *mockMessageStore) Insert(_ context.Context, msg *model.Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.insertCalls++
	if m.insertErr != nil {
		return m.insertErr
	}
	if m.messages == nil {
		m.messages = make(map[int64]*model.Message)
		m.clientSeqs = make(map[string]*model.Message)
	}
	cp := *msg
	m.messages[msg.MsgID] = &cp
	key := fmt.Sprintf("%s|%s|%d", msg.SenderID, msg.SenderSessionID, msg.ClientSeq)
	m.clientSeqs[key] = &cp
	m.allMsgs = append(m.allMsgs, &cp)
	return nil
}

func (m *mockMessageStore) GetByClientSeq(_ context.Context, senderID, sessionID string, clientSeq int64) (*model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := fmt.Sprintf("%s|%s|%d", senderID, sessionID, clientSeq)
	if m.clientSeqs != nil {
		if msg, ok := m.clientSeqs[key]; ok {
			return msg, nil
		}
	}
	return nil, nil
}

// receiptMsgRepo method

func (m *mockMessageStore) Get(_ context.Context, msgID int64) (*model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getCalls++
	if m.getErr != nil {
		return nil, m.getErr
	}
	if m.messages == nil {
		return nil, pgx.ErrNoRows
	}
	msg, ok := m.messages[msgID]
	if !ok {
		return nil, pgx.ErrNoRows
	}
	return msg, nil
}

// syncMsgRepo method

func (m *mockMessageStore) GetMessagesSinceSeq(_ context.Context, convID string, lastSeq int64, limit int) ([]*model.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncCalls++
	if m.syncErr != nil {
		return nil, m.syncErr
	}
	var result []*model.Message
	for _, msg := range m.allMsgs {
		if msg.ConvID == convID && msg.ConvSeq > lastSeq {
			result = append(result, msg)
		}
	}
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

// ---------------------------------------------------------------------------

// mockIDGenerator implements idGenerator.
type mockIDGenerator struct {
	mu     sync.Mutex
	nextID int64
}

func (g *mockIDGenerator) NextID() int64 {
	g.mu.Lock()
	defer g.mu.Unlock()
	id := g.nextID
	g.nextID++
	return id
}

// ---------------------------------------------------------------------------

// mockSeqCache implements seqCache, receiptSeqCache, and syncSeqCache.
type mockSeqCache struct {
	mu          sync.Mutex
	convSeqs    map[string]int64
	userSeqs    map[string]int64
	sessionSeqs map[string]int64
}

// seqCache methods

func (c *mockSeqCache) GetAndIncrementConvSeq(_ context.Context, convID string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.convSeqs == nil {
		c.convSeqs = make(map[string]int64)
	}
	c.convSeqs[convID]++
	return c.convSeqs[convID], nil
}

func (c *mockSeqCache) SetRecentMsg(_ context.Context, convID string, msgID int64, score float64) error {
	return nil
}

// receiptSeqCache methods

func (c *mockSeqCache) SetUserSeq(_ context.Context, userID, convID string, seq int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.userSeqs == nil {
		c.userSeqs = make(map[string]int64)
	}
	key := fmt.Sprintf("%s|%s", userID, convID)
	c.userSeqs[key] = seq
	return nil
}

func (c *mockSeqCache) GetUserSeq(_ context.Context, userID, convID string) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.userSeqs == nil {
		return 0, nil
	}
	key := fmt.Sprintf("%s|%s", userID, convID)
	return c.userSeqs[key], nil
}

// syncSeqCache method

func (c *mockSeqCache) SetSessionSeq(_ context.Context, sessionID, convID string, seq int64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sessionSeqs == nil {
		c.sessionSeqs = make(map[string]int64)
	}
	key := fmt.Sprintf("%s|%s", sessionID, convID)
	c.sessionSeqs[key] = seq
	return nil
}

// ---------------------------------------------------------------------------

// mockConvManager implements convManager, convProvider, and receiptConvRepo.
type mockConvManager struct {
	mu    sync.Mutex
	convs map[string]*model.Conversation
	// members indexed by convID
	members          map[string][]*model.ConvMember
	getOrCreateCalls []struct{ userA, userB string }
}

func (m *mockConvManager) Get(_ context.Context, convID string) (*model.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.convs == nil {
		return nil, model.ErrConvNotFound
	}
	conv, ok := m.convs[convID]
	if !ok {
		return nil, model.ErrConvNotFound
	}
	return conv, nil
}

func (m *mockConvManager) GetOrCreateP2P(_ context.Context, userA, userB string) (*model.Conversation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.getOrCreateCalls = append(m.getOrCreateCalls, struct{ userA, userB string }{userA, userB})
	convID := model.MakeP2PConvID(userA, userB)
	if m.convs == nil {
		m.convs = make(map[string]*model.Conversation)
	}
	if _, exists := m.convs[convID]; !exists {
		m.convs[convID] = &model.Conversation{
			ConvID: convID,
			Type:   model.ConvP2P,
		}
	}
	return m.convs[convID], nil
}

func (m *mockConvManager) IsDirectChatBlocked(_ context.Context, _ string) (bool, error) {
	return false, nil
}

func (m *mockConvManager) GetOrCreateSystemConv(_ context.Context, userID string) (*model.Conversation, error) {
	convID := model.MakeSystemConvID(userID)
	return &model.Conversation{ConvID: convID, Type: model.ConvSystem, OwnerID: userID}, nil
}

func (m *mockConvManager) IsMember(_ context.Context, convID, userID string) (bool, error) {
	return true, nil
}

// convProvider / receiptConvRepo methods

func (m *mockConvManager) GetMembers(_ context.Context, convID string) ([]*model.ConvMember, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.members == nil {
		return nil, nil
	}
	members, ok := m.members[convID]
	if !ok {
		return nil, nil
	}
	return members, nil
}

// ---------------------------------------------------------------------------

// mockSessionGetter implements sessionGetter.
type mockSessionGetter struct {
	mu       sync.Mutex
	sessions map[string][]string
}

func (s *mockSessionGetter) GetUserSessionIDs(_ context.Context, userID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sessions == nil {
		return nil
	}
	return s.sessions[userID]
}

// ---------------------------------------------------------------------------

// mockConnRegistry implements connRegistry, connBySessionID, and receiptConnRegistry.
type mockConnRegistry struct {
	mu          sync.Mutex
	bySessionID map[string]any
	byUserID    map[string][]any
}

func (r *mockConnRegistry) GetBySessionID(_ context.Context, sessionID string) any {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.bySessionID == nil {
		return nil
	}
	return r.bySessionID[sessionID]
}

func (r *mockConnRegistry) GetByUserID(_ context.Context, userID string) []any {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byUserID == nil {
		return nil
	}
	return r.byUserID[userID]
}

// ---------------------------------------------------------------------------

// mockReceiptWriter implements receiptWriter.
type mockContactCreator struct{}

func (m *mockContactCreator) AddContact(_ context.Context, _, _ string) error { return nil }

type mockUserGetter struct{}

func (m *mockUserGetter) GetByID(_ context.Context, id string) (*model.User, error) {
	return &model.User{Name: id}, nil
}

type mockWhForwarder struct{}

func (m *mockWhForwarder) ListByConvID(_ context.Context, _ string) ([]*model.ConvWebhook, error) {
	return nil, nil
}
func (m *mockWhForwarder) GetByConvIDAndName(_ context.Context, _, _ string) (*model.ConvWebhook, error) {
	return nil, nil
}

type mockContactRequestDB struct{}

func (m *mockContactRequestDB) GetByFormMsgID(_ context.Context, _ int64) (*model.ContactRequest, error) {
	return nil, nil
}
func (m *mockContactRequestDB) GetByID(_ context.Context, _ int64) (*model.ContactRequest, error) {
	return nil, nil
}
func (m *mockContactRequestDB) Insert(_ context.Context, _ *model.ContactRequest) (int64, error) {
	return 1, nil
}
func (m *mockContactRequestDB) UpdateStatus(_ context.Context, _ int64, _ model.ContactRequestStatus) error {
	return nil
}
func (m *mockContactRequestDB) UpdateStatusTx(_ context.Context, _ pgx.Tx, _ int64, _ model.ContactRequestStatus) error {
	return nil
}
func (m *mockContactRequestDB) LockByIDTx(_ context.Context, _ pgx.Tx, _ int64) (*model.ContactRequest, error) {
	return nil, nil
}
func (m *mockContactRequestDB) UpdateFormMsgID(_ context.Context, _, _ int64) error {
	return nil
}
func (m *mockContactRequestDB) Delete(_ context.Context, _ int64) error {
	return nil
}

type mockReceiptWriter struct {
	mu        sync.Mutex
	receipts  []*model.Receipt
	upsertErr error
}

func (w *mockReceiptWriter) Upsert(_ context.Context, rc *model.Receipt) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.receipts = append(w.receipts, rc)
	if w.upsertErr != nil {
		return w.upsertErr
	}
	return nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newIngestFixture creates an Ingest with mock dependencies. The caller can
// further customize mocks before calling ingest.Ingest.
//   - ratePerSec, burst, maxBody: RateLimiter parameters.
//   - defaultID: starting value for the ID generator.
func newIngestFixture(ratePerSec, burst, maxBody int, defaultID int64) (
	ing *Ingest,
	store *mockMessageStore,
	idGen *mockIDGenerator,
	seqCache *mockSeqCache,
	convMgr *mockConvManager,
	sessGtr *mockSessionGetter,
	connReg *mockConnRegistry,
	receiptW *mockReceiptWriter,
	rateLmt *RateLimiter,
) {
	store = &mockMessageStore{}
	idGen = &mockIDGenerator{nextID: defaultID}
	seqCache = &mockSeqCache{}
	convMgr = &mockConvManager{}
	sessGtr = &mockSessionGetter{}
	connReg = &mockConnRegistry{}
	receiptW = &mockReceiptWriter{}
	rateLmt = NewRateLimiter(ratePerSec, burst, maxBody)
	router := NewRouter(sessGtr, convMgr, connReg)
	pusher := NewPusher(connReg, receiptW)
	contactReqDB := &mockContactRequestDB{}
	ing = NewIngest(store, router, pusher, rateLmt, idGen, seqCache, convMgr, contactReqDB, &mockContactCreator{}, &mockUserGetter{}, &mockWhForwarder{}, "")
	return
}

// defaultRouterFixture creates a Router pre-configured with:
//   - A P2P conversation "user_a:user_b" with members [user_a, user_b].
//   - user_b has sessions [sess_b1, sess_b2].
//
// The caller can override fields on the returned mocks.
func defaultRouterFixture() (*Router, *mockConvManager, *mockSessionGetter) {
	convMgr := &mockConvManager{
		convs: map[string]*model.Conversation{
			"user_a:user_b": {ConvID: "user_a:user_b", Type: model.ConvP2P},
		},
		members: map[string][]*model.ConvMember{
			"user_a:user_b": {
				{ConvID: "user_a:user_b", UserID: "user_a"},
				{ConvID: "user_a:user_b", UserID: "user_b"},
			},
		},
	}
	sessGtr := &mockSessionGetter{
		sessions: map[string][]string{
			"user_b": {"sess_b1", "sess_b2"},
		},
	}
	router := NewRouter(sessGtr, convMgr, &mockConnRegistry{})
	return router, convMgr, sessGtr
}

// defaultRouterGroupFixture creates a Router pre-configured with a group
// conversation "group_1" with three members.
func defaultRouterGroupFixture() (*Router, *mockConvManager, *mockSessionGetter) {
	convMgr := &mockConvManager{
		convs: map[string]*model.Conversation{
			"group_1": {ConvID: "group_1", Type: model.ConvGroup},
		},
		members: map[string][]*model.ConvMember{
			"group_1": {
				{ConvID: "group_1", UserID: "user_a"},
				{ConvID: "group_1", UserID: "user_b"},
				{ConvID: "group_1", UserID: "user_c"},
			},
		},
	}
	sessGtr := &mockSessionGetter{
		sessions: map[string][]string{
			"user_b": {"sess_b1"},
			"user_c": {"sess_c1", "sess_c2"},
		},
	}
	router := NewRouter(sessGtr, convMgr, &mockConnRegistry{})
	return router, convMgr, sessGtr
}

// basicMsg returns a simple model.Message for use in Router / Pusher tests.
func basicMsg() *model.Message {
	return &model.Message{
		MsgID:     1001,
		ConvID:    "user_a:user_b",
		SenderID:  "user_a",
		Body:      "hello",
		Timestamp: time.Now().UnixMilli(),
		ConvSeq:   5,
	}
}

// ---------------------------------------------------------------------------
// parseP2PCounterpart
// ---------------------------------------------------------------------------

func TestParseP2PCounterpart(t *testing.T) {
	cases := []struct {
		name     string
		convID   string
		senderID string
		want     string
	}{
		{name: "sender is parts[0]", convID: "user_a:user_b", senderID: "user_a", want: "user_b"},
		{name: "sender is parts[1]", convID: "user_a:user_b", senderID: "user_b", want: "user_a"},
		{name: "three segments splits at first colon", convID: "a:b:c", senderID: "a", want: "b:c"},
		{name: "no colon returns empty", convID: "justastring", senderID: "user_a", want: ""},
		{name: "empty string", convID: "", senderID: "user_a", want: ""},
		{name: "trailing colon returns empty second part", convID: "user_a:", senderID: "user_a", want: ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseP2PCounterpart(tc.convID, tc.senderID)
			if got != tc.want {
				t.Errorf("parseP2PCounterpart(%q, %q) = %q; want %q", tc.convID, tc.senderID, got, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Ingest
// ---------------------------------------------------------------------------

func TestIngest_Success(t *testing.T) {
	ing, store, idGen, seqCache, convMgr, sessGtr, connReg, receiptW, _ := newIngestFixture(100, 100, 100000, 100)

	// Set up a P2P conversation that already exists.
	convMgr.convs = map[string]*model.Conversation{
		"user_a:user_b": {ConvID: "user_a:user_b", Type: model.ConvP2P},
	}
	// Set up the target user's sessions and members for routing.
	sessGtr.sessions = map[string][]string{"user_b": {"sess_b1"}}
	convMgr.members = map[string][]*model.ConvMember{
		"user_a:user_b": {
			{ConvID: "user_a:user_b", UserID: "user_a"},
			{ConvID: "user_a:user_b", UserID: "user_b"},
		},
	}
	// Provide a live connection for the target session.
	conn := &mockConn{}
	connReg.bySessionID = map[string]any{"sess_b1": conn}

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "hello world",
		ClientSeq:   42,
	}
	ack, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	// Verify ack fields.
	if ack.MsgID != 100 {
		t.Errorf("ack.MsgID = %d; want 100", ack.MsgID)
	}
	if ack.ClientSeq != 42 {
		t.Errorf("ack.ClientSeq = %d; want 42", ack.ClientSeq)
	}
	if ack.Status != int(model.MsgSent) {
		t.Errorf("ack.Status = %d; want %d", ack.Status, model.MsgSent)
	}
	if ack.Timestamp == 0 {
		t.Errorf("ack.Timestamp = 0; want non-zero")
	}

	// Verify message was persisted.
	if store.insertCalls != 1 {
		t.Errorf("store.Insert calls = %d; want 1", store.insertCalls)
	}
	stored := store.messages[100]
	if stored == nil {
		t.Fatal("stored message not found by MsgID 100")
	}
	if stored.ConvID != "user_a:user_b" {
		t.Errorf("stored.ConvID = %q; want user_a:user_b", stored.ConvID)
	}
	if stored.SenderID != "user_a" {
		t.Errorf("stored.SenderID = %q; want user_a", stored.SenderID)
	}
	if stored.ConvSeq != 1 {
		t.Errorf("stored.ConvSeq = %d; want 1", stored.ConvSeq)
	}
	if stored.Body != "hello world" {
		t.Errorf("stored.Body = %q; want hello world", stored.Body)
	}

	// Verify next ID was consumed.
	if idGen.nextID != 101 {
		t.Errorf("idGen.nextID = %d; want 101", idGen.nextID)
	}

	// Verify conv seq was incremented.
	if seqCache.convSeqs["user_a:user_b"] != 1 {
		t.Errorf("conv seq = %d; want 1", seqCache.convSeqs["user_a:user_b"])
	}

	// Verify push was sent.
	frames := conn.getFrames()
	if len(frames) != 1 {
		t.Fatalf("pusher sent %d frames; want 1", len(frames))
	}
	if frames[0].Type != protocol.MsgPush {
		t.Errorf("frame type = %d; want %d", frames[0].Type, protocol.MsgPush)
	}

	// Verify delivery receipt was written.
	if len(receiptW.receipts) != 1 {
		t.Fatalf("receipts written = %d; want 1", len(receiptW.receipts))
	}
	rc := receiptW.receipts[0]
	if rc.MsgID != 100 {
		t.Errorf("receipt.MsgID = %d; want 100", rc.MsgID)
	}
	if rc.UserID != "user_b" {
		t.Errorf("receipt.UserID = %s; want user_b", rc.UserID)
	}
	if rc.Status != model.ReceiptDelivered {
		t.Errorf("receipt.Status = %d; want %d", rc.Status, model.ReceiptDelivered)
	}
}

func TestIngest_RateLimited(t *testing.T) {
	// burst=0 means the first Check call always returns ErrRateLimited.
	ing, _, _, _, _, _, _, _, _ := newIngestFixture(0, 0, 100000, 100)

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "hello",
		ClientSeq:   1,
	}
	_, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err == nil {
		t.Fatal("expected rate-limit error, got nil")
	}
	if !errors.Is(err, model.ErrRateLimited) {
		t.Errorf("error = %v; want ErrRateLimited", err)
	}
}

func TestIngest_BodyTooLarge(t *testing.T) {
	// maxBody=5 reject bodies longer than 5 bytes.
	ing, _, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 5, 100)

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "hello world", // 11 chars > 5
		ClientSeq:   1,
	}
	_, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err == nil {
		t.Fatal("expected body-too-large error, got nil")
	}
	if !errors.Is(err, model.ErrMsgTooLarge) {
		t.Errorf("error = %v; want ErrMsgTooLarge", err)
	}
}

func TestIngest_Duplicate(t *testing.T) {
	ing, store, _, _, convMgr, _, _, _, _ := newIngestFixture(100, 100, 100000, 200)

	convMgr.convs = map[string]*model.Conversation{
		"user_a:user_b": {ConvID: "user_a:user_b", Type: model.ConvP2P},
	}

	// Pre-insert a message that the dedup check will find.
	existingMsg := &model.Message{
		MsgID:     999,
		ConvID:    "user_a:user_b",
		SenderID:  "user_a",
		ClientSeq: 7,
		Timestamp: 1000000,
		Status:    model.MsgSent,
	}
	store.messages = map[int64]*model.Message{999: existingMsg}
	store.clientSeqs = map[string]*model.Message{
		fmt.Sprintf("%s|%s|%d", "user_a", "sess_a1", int64(7)): existingMsg,
	}

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "dup",
		ClientSeq:   7, // same clientSeq as existing
	}
	ack, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}

	// Must return ack with the existing MsgID (999) not a new ID.
	if ack.MsgID != 999 {
		t.Errorf("ack.MsgID = %d; want 999 (existing MsgID)", ack.MsgID)
	}
	if ack.ClientSeq != 7 {
		t.Errorf("ack.ClientSeq = %d; want 7", ack.ClientSeq)
	}
	if ack.Status != int(model.MsgSent) {
		t.Errorf("ack.Status = %d; want %d", ack.Status, model.MsgSent)
	}

	// Must NOT persist a new message (insertCalls should stay 0).
	if store.insertCalls != 0 {
		t.Errorf("store.Insert was called %d times; want 0 (dedup should skip)", store.insertCalls)
	}
}

func TestIngest_AutoCreateP2P(t *testing.T) {
	ing, store, _, _, convMgr, sessGtr, connReg, receiptW, _ := newIngestFixture(100, 100, 100000, 100)

	// NO pre-existing conversation - Get will return ErrConvNotFound.
	// convID "user_a:user_b" will pass IsP2PConvID.
	// Pre-set members so the Router can route after auto-creation.
	convMgr.members = map[string][]*model.ConvMember{
		"user_a:user_b": {
			{ConvID: "user_a:user_b", UserID: "user_a"},
			{ConvID: "user_a:user_b", UserID: "user_b"},
		},
	}
	sessGtr.sessions = map[string][]string{"user_b": {"sess_b1"}}
	conn := &mockConn{}
	connReg.bySessionID = map[string]any{"sess_b1": conn}

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "first msg",
		ClientSeq:   1,
	}
	ack, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err != nil {
		t.Fatalf("Ingest returned error: %v", err)
	}
	if ack.MsgID != 100 {
		t.Errorf("ack.MsgID = %d; want 100", ack.MsgID)
	}

	// Verify GetOrCreateP2P was called.
	if len(convMgr.getOrCreateCalls) != 1 {
		t.Fatalf("GetOrCreateP2P calls = %d; want 1", len(convMgr.getOrCreateCalls))
	}
	call := convMgr.getOrCreateCalls[0]
	if call.userA != "user_a" {
		t.Errorf("GetOrCreateP2P userA = %q; want user_a", call.userA)
	}
	if call.userB != "user_b" {
		t.Errorf("GetOrCreateP2P userB = %q; want user_b", call.userB)
	}

	// Verify the conversation was created.
	conv, err := convMgr.Get(context.Background(), "user_a:user_b")
	if err != nil {
		t.Fatalf("expected conv to exist after GetOrCreateP2P: %v", err)
	}
	if conv.Type != model.ConvP2P {
		t.Errorf("conv.Type = %d; want %d", conv.Type, model.ConvP2P)
	}

	// Message should have been persisted and pushed.
	if store.insertCalls != 1 {
		t.Errorf("store.Insert calls = %d; want 1", store.insertCalls)
	}
	if len(receiptW.receipts) != 1 {
		t.Errorf("receipts written = %d; want 1", len(receiptW.receipts))
	}
}

func TestIngest_AutoCreateP2P_EmptyOtherID(t *testing.T) {
	ing, _, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)

	// Using an invalid P2P convID that still passes IsP2PConvID but
	// has no counterpart (empty after ":").
	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:", // passes IsP2PConvID but counterpart is ""
		ContentType: 0,
		Body:        "hello",
		ClientSeq:   1,
	}
	_, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err == nil {
		t.Fatal("expected ErrConvNotFound for empty counterpart")
	}
	if !errors.Is(err, model.ErrConvNotFound) {
		t.Errorf("error = %v; want ErrConvNotFound", err)
	}
}

func TestIngest_GroupConvNotFound(t *testing.T) {
	ing, _, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)

	payload := protocol.MsgSendPayload{
		ConvID:      "group_999",
		ContentType: 0,
		Body:        "hello",
		ClientSeq:   1,
	}
	_, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err == nil {
		t.Fatal("expected ErrConvNotFound for non-existent group conv")
	}
	if !errors.Is(err, model.ErrConvNotFound) {
		t.Errorf("error = %v; want ErrConvNotFound", err)
	}
}

func TestIngest_InsertError(t *testing.T) {
	ing, store, _, _, convMgr, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)

	convMgr.convs = map[string]*model.Conversation{
		"user_a:user_b": {ConvID: "user_a:user_b", Type: model.ConvP2P},
	}
	store.insertErr = errors.New("db down")

	payload := protocol.MsgSendPayload{
		ConvID:      "user_a:user_b",
		ContentType: 0,
		Body:        "hello",
		ClientSeq:   1,
	}
	_, err := ing.Ingest(context.Background(), "user_a", "sess_a1", payload)
	if err == nil {
		t.Fatal("expected insert error")
	}
	if err.Error() != "db down" {
		t.Errorf("error = %v; want db down", err)
	}
}

func TestSendSystemMessage(t *testing.T) {
	ing, store, idGen, seqCache, convMgr, sessGtr, connReg, receiptW, _ := newIngestFixture(100, 100, 100000, 100)

	// Provide routing support so the system message gets pushed.
	convMgr.convs = map[string]*model.Conversation{
		"conv_sys": {ConvID: "conv_sys", Type: model.ConvP2P},
	}
	convMgr.members = map[string][]*model.ConvMember{
		"conv_sys": {
			{ConvID: "conv_sys", UserID: "user_a"},
			{ConvID: "conv_sys", UserID: "user_b"},
		},
	}
	sessGtr.sessions = map[string][]string{"user_b": {"sess_b1"}}
	conn := &mockConn{}
	connReg.bySessionID = map[string]any{"sess_b1": conn}

	msg, err := ing.SendSystemMessage(context.Background(), "conv_sys", "system notice")
	if err != nil {
		t.Fatalf("SendSystemMessage error: %v", err)
	}

	// Verify returned message fields.
	if msg.MsgID != 100 {
		t.Errorf("msg.MsgID = %d; want 100", msg.MsgID)
	}
	if msg.ConvID != "conv_sys" {
		t.Errorf("msg.ConvID = %q; want conv_sys", msg.ConvID)
	}
	if msg.SenderID != "system" {
		t.Errorf("msg.SenderID = %q; want system", msg.SenderID)
	}
	if msg.ContentType != model.ContentSystem {
		t.Errorf("msg.ContentType = %d; want %d", msg.ContentType, model.ContentSystem)
	}
	if msg.Body != "system notice" {
		t.Errorf("msg.Body = %q; want system notice", msg.Body)
	}
	if msg.Status != model.MsgSent {
		t.Errorf("msg.Status = %d; want %d", msg.Status, model.MsgSent)
	}
	if msg.Timestamp == 0 {
		t.Errorf("msg.Timestamp = 0; want non-zero")
	}

	// Verify ID generator.
	if idGen.nextID != 101 {
		t.Errorf("idGen.nextID = %d; want 101", idGen.nextID)
	}

	// Verify conv seq.
	if seqCache.convSeqs["conv_sys"] != 1 {
		t.Errorf("conv seq = %d; want 1", seqCache.convSeqs["conv_sys"])
	}

	// Verify persisted.
	if store.insertCalls != 1 {
		t.Errorf("store.Insert calls = %d; want 1", store.insertCalls)
	}
	stored := store.messages[100]
	if stored == nil {
		t.Fatal("stored message not found")
	}
	if stored.Body != "system notice" {
		t.Errorf("stored.Body = %q; want system notice", stored.Body)
	}

	// Verify push.
	frames := conn.getFrames()
	if len(frames) != 1 {
		t.Fatalf("pusher sent %d frames; want 1", len(frames))
	}
	if frames[0].Type != protocol.MsgPush {
		t.Errorf("frame type = %d; want %d", frames[0].Type, protocol.MsgPush)
	}

	// Verify receipt.
	if len(receiptW.receipts) != 1 {
		t.Fatalf("receipts = %d; want 1", len(receiptW.receipts))
	}
	if receiptW.receipts[0].MsgID != 100 {
		t.Errorf("receipt.MsgID = %d; want 100", receiptW.receipts[0].MsgID)
	}
}

func TestSendSystemMessage_InsertError(t *testing.T) {
	ing, store, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)
	store.insertErr = errors.New("db error")

	_, err := ing.SendSystemMessage(context.Background(), "conv_sys", "body")
	if err == nil {
		t.Fatal("expected error from failed insert")
	}
}

// ---------------------------------------------------------------------------
// Router
// ---------------------------------------------------------------------------

func TestRouter_P2P(t *testing.T) {
	router, _, _ := defaultRouterFixture()

	msg := basicMsg()
	targets := router.Route(context.Background(), msg)
	if len(targets) != 1 {
		t.Fatalf("Route returned %d targets; want 1", len(targets))
	}
	target := targets[0]
	if target.UserID != "user_b" {
		t.Errorf("target.UserID = %q; want user_b", target.UserID)
	}
	if len(target.SessionIDs) != 2 {
		t.Fatalf("target.SessionIDs = %v; want [sess_b1, sess_b2]", target.SessionIDs)
	}
	if target.SessionIDs[0] != "sess_b1" {
		t.Errorf("SessionIDs[0] = %q; want sess_b1", target.SessionIDs[0])
	}
	if target.SessionIDs[1] != "sess_b2" {
		t.Errorf("SessionIDs[1] = %q; want sess_b2", target.SessionIDs[1])
	}
}

func TestRouter_P2P_SkipsSender(t *testing.T) {
	router, convMgr, sessGtr := defaultRouterFixture()

	// Give user_a some sessions so the route target is not dropped.
	sessGtr.sessions = map[string][]string{
		"user_b": {"sess_b1", "sess_b2"},
		"user_a": {"sess_a1"},
	}

	// user_b sends the message; user_b's non-sending sessions and user_a should be targets.
	msg := basicMsg()
	msg.SenderID = "user_b"
	targets := router.Route(context.Background(), msg)
	if len(targets) != 2 {
		t.Fatalf("Route returned %d targets; want 2", len(targets))
	}
	byUser := make(map[string][]string)
	for _, tr := range targets {
		byUser[tr.UserID] = tr.SessionIDs
	}
	if _, ok := byUser["user_a"]; !ok {
		t.Errorf("user_a should be a target")
	}
	// user_b's other sessions should still receive the push
	sessions, ok := byUser["user_b"]
	if !ok {
		t.Errorf("user_b should be a target (other sessions)")
	} else if len(sessions) != 2 {
		t.Errorf("user_b sessions = %v; want [sess_b1, sess_b2]", sessions)
	}
	_ = convMgr
}

func TestRouter_Group(t *testing.T) {
	router, _, _ := defaultRouterGroupFixture()

	msg := basicMsg()
	msg.ConvID = "group_1"
	msg.SenderID = "user_a"
	targets := router.Route(context.Background(), msg)
	if len(targets) != 2 {
		t.Fatalf("Route returned %d targets; want 2", len(targets))
	}

	// Build a set of targets for easy assertion.
	byUser := make(map[string][]string)
	for _, tr := range targets {
		byUser[tr.UserID] = tr.SessionIDs
	}

	if _, ok := byUser["user_a"]; ok {
		t.Errorf("sender user_a should not appear in targets")
	}
	if sessions, ok := byUser["user_b"]; !ok {
		t.Errorf("user_b should be a target")
	} else if len(sessions) != 1 || sessions[0] != "sess_b1" {
		t.Errorf("user_b sessions = %v; want [sess_b1]", sessions)
	}
	if sessions, ok := byUser["user_c"]; !ok {
		t.Errorf("user_c should be a target")
	} else if len(sessions) != 2 || sessions[0] != "sess_c1" || sessions[1] != "sess_c2" {
		t.Errorf("user_c sessions = %v; want [sess_c1, sess_c2]", sessions)
	}
}

func TestRouter_Group_SkipsSender(t *testing.T) {
	router, convMgr, sessGtr := defaultRouterGroupFixture()

	// Give user_a sessions so the Router includes them as a target.
	sessGtr.sessions["user_a"] = []string{"sess_a1"}

	// user_b sends; user_a, user_c, and user_b's other sessions should be targets.
	msg := basicMsg()
	msg.ConvID = "group_1"
	msg.SenderID = "user_b"
	targets := router.Route(context.Background(), msg)

	byUser := make(map[string]bool)
	for _, tr := range targets {
		byUser[tr.UserID] = true
	}
	if !byUser["user_b"] {
		t.Errorf("user_b should be a target (other sessions)")
	}
	if !byUser["user_a"] {
		t.Errorf("user_a should be a target")
	}
	if !byUser["user_c"] {
		t.Errorf("user_c should be a target")
	}
	_ = convMgr // used
}

func TestRouter_ConvNotFound(t *testing.T) {
	router, convMgr, _ := defaultRouterFixture()
	delete(convMgr.convs, "user_a:user_b")

	msg := basicMsg()
	targets := router.Route(context.Background(), msg)
	if targets != nil {
		t.Errorf("Route returned %d targets; want nil", len(targets))
	}
}

func TestRouter_NoSessionsForTarget(t *testing.T) {
	router, convMgr, sessGtr := defaultRouterFixture()

	// Remove sessions for user_b – the only target.
	sessGtr.sessions = map[string][]string{}

	// Add a member with no sessions at all.
	convMgr.members["user_a:user_b"] = []*model.ConvMember{
		{ConvID: "user_a:user_b", UserID: "user_a"},
		{ConvID: "user_a:user_b", UserID: "user_b"},
	}

	msg := basicMsg()
	targets := router.Route(context.Background(), msg)
	if len(targets) != 0 {
		t.Errorf("Route returned %d targets; want 0 (no sessions for target)", len(targets))
	}
}

func TestRouter_GetMembersError(t *testing.T) {
	router, convMgr, _ := defaultRouterFixture()
	// Trigger an error by removing the members entry for the conv.
	delete(convMgr.members, "user_a:user_b")

	msg := basicMsg()
	targets := router.Route(context.Background(), msg)
	if targets != nil {
		t.Errorf("Route returned %d targets; want nil on GetMembers error", len(targets))
	}
}

// ---------------------------------------------------------------------------
// Pusher
// ---------------------------------------------------------------------------

func TestPush_Success(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	pusher := NewPusher(connReg, receiptW)

	connB1 := &mockConn{}
	connB2 := &mockConn{}
	connReg.bySessionID = map[string]any{
		"sess_b1": connB1,
		"sess_b2": connB2,
	}

	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{"sess_b1", "sess_b2"}},
	}
	pusher.Push(context.Background(), msg, targets)

	// Each session should have received exactly one MsgPush frame.
	if len(connB1.getFrames()) != 1 {
		t.Errorf("connB1 frames = %d; want 1", len(connB1.getFrames()))
	}
	if len(connB2.getFrames()) != 1 {
		t.Errorf("connB2 frames = %d; want 1", len(connB2.getFrames()))
	}
	frame := connB1.getFrames()[0]
	if frame.Type != protocol.MsgPush {
		t.Errorf("frame.Type = %d; want %d", frame.Type, protocol.MsgPush)
	}

	// Delivery receipts: one per session.
	if len(receiptW.receipts) != 2 {
		t.Fatalf("receipts = %d; want 2", len(receiptW.receipts))
	}
	for _, rc := range receiptW.receipts {
		if rc.MsgID != msg.MsgID {
			t.Errorf("receipt.MsgID = %d; want %d", rc.MsgID, msg.MsgID)
		}
		if rc.UserID != "user_b" {
			t.Errorf("receipt.UserID = %q; want user_b", rc.UserID)
		}
		if rc.Status != model.ReceiptDelivered {
			t.Errorf("receipt.Status = %d; want %d", rc.Status, model.ReceiptDelivered)
		}
		if rc.SessionID != "sess_b1" && rc.SessionID != "sess_b2" {
			t.Errorf("receipt.SessionID = %q; want sess_b1 or sess_b2", rc.SessionID)
		}
	}
}

func TestPush_NilConnection(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	pusher := NewPusher(connReg, receiptW)

	// bySessionID is empty, so GetBySessionID returns nil.
	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{"sess_b1"}},
	}
	pusher.Push(context.Background(), msg, targets)

	// No frames sent, no receipts.
	if len(receiptW.receipts) != 0 {
		t.Errorf("receipts = %d; want 0", len(receiptW.receipts))
	}
}

func TestPush_NoSessionIDs(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	pusher := NewPusher(connReg, receiptW)

	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{}}, // empty
	}
	pusher.Push(context.Background(), msg, targets)

	// No sessions to push to.
	if len(receiptW.receipts) != 0 {
		t.Errorf("receipts = %d; want 0", len(receiptW.receipts))
	}
}

func TestPush_SendFrameError(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	pusher := NewPusher(connReg, receiptW)

	errConn := &mockConn{sendErr: errors.New("write failed")}
	goodConn := &mockConn{}
	connReg.bySessionID = map[string]any{
		"sess_err":  errConn,
		"sess_good": goodConn,
	}

	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{"sess_err", "sess_good"}},
	}
	// Should not panic.
	pusher.Push(context.Background(), msg, targets)

	// errConn's SendFrame error should skip receipt for that session.
	if len(errConn.getFrames()) != 1 {
		t.Errorf("errConn sent %d frames; want 1 (SendFrame called despite error)", len(errConn.getFrames()))
	}
	if len(goodConn.getFrames()) != 1 {
		t.Errorf("goodConn sent %d frames; want 1", len(goodConn.getFrames()))
	}

	// Only the successful send gets a receipt.
	if len(receiptW.receipts) != 1 {
		t.Fatalf("receipts = %d; want 1 (only good session)", len(receiptW.receipts))
	}
	if receiptW.receipts[0].SessionID != "sess_good" {
		t.Errorf("receipt.SessionID = %q; want sess_good", receiptW.receipts[0].SessionID)
	}
}

func TestPush_ConnTypeAssertionFails(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	pusher := NewPusher(connReg, receiptW)

	// Store a plain struct that does NOT implement SendFrame.
	connReg.bySessionID = map[string]any{
		"sess_no_send": "not-a-connection",
	}

	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{"sess_no_send"}},
	}
	// Should not panic.
	pusher.Push(context.Background(), msg, targets)

	if len(receiptW.receipts) != 0 {
		t.Errorf("receipts = %d; want 0 (type assertion failed)", len(receiptW.receipts))
	}
}

func TestPush_ReceiptUpsertError(t *testing.T) {
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{upsertErr: errors.New("upsert failed")}
	pusher := NewPusher(connReg, receiptW)

	conn := &mockConn{}
	connReg.bySessionID = map[string]any{"sess_b1": conn}

	msg := basicMsg()
	targets := []RouteTarget{
		{UserID: "user_b", SessionIDs: []string{"sess_b1"}},
	}
	pusher.Push(context.Background(), msg, targets)

	// Frame should still be sent.
	if len(conn.getFrames()) != 1 {
		t.Errorf("conn sent %d frames; want 1 (push should proceed despite receipt error)", len(conn.getFrames()))
	}
	// Receipt should be captured (the mock still records it even when returning an error).
	if len(receiptW.receipts) != 1 {
		t.Errorf("receipts = %d; want 1", len(receiptW.receipts))
	}
}

// ---------------------------------------------------------------------------
// ReceiptHandler
// ---------------------------------------------------------------------------

func TestMarkRead_Success(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}

	// Pre-populate a message with a different sender.
	msg := &model.Message{
		MsgID:    200,
		ConvID:   "conv_x",
		SenderID: "user_b",
		ConvSeq:  50,
	}
	store.messages = map[int64]*model.Message{200: msg}

	// Sender has connections.
	senderConn := &mockConn{}
	connReg.byUserID = map[string][]any{"user_b": {senderConn}}

	handler := NewReceiptHandler(store, seqCache, nil, connReg, receiptW)

	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 200)
	if err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	// User seq was set.
	if seqCache.userSeqs["user_a|conv_x"] != 50 {
		t.Errorf("user seq = %d; want 50", seqCache.userSeqs["user_a|conv_x"])
	}

	// Read notify was sent to sender's connections.
	frames := senderConn.getFrames()
	if len(frames) != 1 {
		t.Fatalf("sender conn frames = %d; want 1", len(frames))
	}
	if frames[0].Type != protocol.MsgReadNotify {
		t.Errorf("frame.Type = %d; want %d", frames[0].Type, protocol.MsgReadNotify)
	}

	// Receipt was written.
	if len(receiptW.receipts) != 1 {
		t.Fatalf("receipts = %d; want 1", len(receiptW.receipts))
	}
	rc := receiptW.receipts[0]
	if rc.MsgID != 200 {
		t.Errorf("receipt.MsgID = %d; want 200", rc.MsgID)
	}
	if rc.UserID != "user_a" {
		t.Errorf("receipt.UserID = %q; want user_a", rc.UserID)
	}
	if rc.Status != model.ReceiptRead {
		t.Errorf("receipt.Status = %d; want %d", rc.Status, model.ReceiptRead)
	}
}

func TestMarkRead_MsgNotFound(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	receiptW := &mockReceiptWriter{}

	handler := NewReceiptHandler(store, seqCache, nil, nil, receiptW)

	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 999)
	if err != nil {
		t.Fatalf("MarkRead should return nil when msg not found, got: %v", err)
	}

	// User seq should NOT be set (early return before SetUserSeq).
	if seqCache.userSeqs["user_a|conv_x"] != 0 {
		t.Errorf("user seq = %d; want 0", seqCache.userSeqs["user_a|conv_x"])
	}

	// No receipt should be written (msg not found → no notify → no receipt).
	if len(receiptW.receipts) != 0 {
		t.Errorf("receipts = %d; want 0", len(receiptW.receipts))
	}
}

func TestMarkRead_OwnMessage(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	receiptW := &mockReceiptWriter{}

	// Message was sent by user_a (same as the reader).
	store.messages = map[int64]*model.Message{
		101: {MsgID: 101, ConvID: "conv_x", SenderID: "user_a", ConvSeq: 30},
	}

	handler := NewReceiptHandler(store, seqCache, nil, nil, receiptW)

	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 101)
	if err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	// User seq should be set.
	if seqCache.userSeqs["user_a|conv_x"] != 30 {
		t.Errorf("user seq = %d; want 30", seqCache.userSeqs["user_a|conv_x"])
	}

	// No receipt should be written for own message.
	if len(receiptW.receipts) != 0 {
		t.Errorf("receipts = %d; want 0 (own message)", len(receiptW.receipts))
	}
}

func TestMarkRead_NilReceiptWriter(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	connReg := &mockConnRegistry{}

	msg := &model.Message{MsgID: 300, ConvID: "conv_x", SenderID: "user_b", ConvSeq: 60}
	store.messages = map[int64]*model.Message{300: msg}

	senderConn := &mockConn{}
	connReg.byUserID = map[string][]any{"user_b": {senderConn}}

	// receipt writer is nil.
	handler := NewReceiptHandler(store, seqCache, nil, connReg, nil)

	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 300)
	if err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	// User seq was set.
	if seqCache.userSeqs["user_a|conv_x"] != 60 {
		t.Errorf("user seq = %d; want 60", seqCache.userSeqs["user_a|conv_x"])
	}

	// Read notify was still sent.
	frames := senderConn.getFrames()
	if len(frames) != 1 {
		t.Fatalf("sender conn frames = %d; want 1", len(frames))
	}
}

// errSeqCache implements receiptSeqCache; all methods return an error.
type errSeqCache struct{ err error }

func (e *errSeqCache) SetUserSeq(_ context.Context, _, _ string, _ int64) error {
	return e.err
}

func (e *errSeqCache) GetUserSeq(_ context.Context, _, _ string) (int64, error) {
	return 0, e.err
}

func (e *errSeqCache) GetAndIncrementConvSeq(_ context.Context, _ string) (int64, error) {
	return 0, e.err
}

func TestMarkRead_SetUserSeqError(t *testing.T) {
	failSeq := &errSeqCache{err: errors.New("seq set failed")}
	store := &mockMessageStore{
		messages: map[int64]*model.Message{400: {MsgID: 400, ConvID: "conv_x", SenderID: "user_b", ConvSeq: 70}},
	}
	receiptW := &mockReceiptWriter{}

	handler := NewReceiptHandler(store, failSeq, nil, nil, receiptW)
	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 400)
	if err == nil {
		t.Fatal("expected error from SetUserSeq failure, got nil")
	}
}

func TestMarkRead_SenderConnTypeAssertionFails(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}

	store.messages = map[int64]*model.Message{
		500: {MsgID: 500, ConvID: "conv_x", SenderID: "user_b"},
	}
	// Connection does NOT implement SendFrame.
	connReg.byUserID = map[string][]any{"user_b": {"plain-string"}}

	handler := NewReceiptHandler(store, seqCache, nil, connReg, receiptW)

	err := handler.MarkRead(context.Background(), "user_a", "conv_x", 500)
	if err != nil {
		t.Fatalf("MarkRead error: %v", err)
	}

	// Receipt should still be written even if notify send fails.
	if len(receiptW.receipts) != 1 {
		t.Errorf("receipts = %d; want 1", len(receiptW.receipts))
	}
}

// ---------------------------------------------------------------------------
// SyncHandler
// ---------------------------------------------------------------------------

func TestSync_Success(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	// Insert a few messages for the conversation.
	insertSyncMessages(store, "conv_y", 1, 3) // ConvSeq 1,2,3

	req := protocol.SyncReqPayload{
		ConvID:      "conv_y",
		LastConvSeq: 0,
		Limit:       10,
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	if res.ConvID != "conv_y" {
		t.Errorf("res.ConvID = %q; want conv_y", res.ConvID)
	}
	if len(res.Messages) != 3 {
		t.Fatalf("len(Messages) = %d; want 3", len(res.Messages))
	}
	if res.HasMore {
		t.Errorf("HasMore = true; want false")
	}

	// Verify message content.
	for i, sm := range res.Messages {
		wantSeq := int64(i + 1)
		if sm.ConvSeq != wantSeq {
			t.Errorf("Messages[%d].ConvSeq = %d; want %d", i, sm.ConvSeq, wantSeq)
		}
	}

	// Session seq was set to the last ConvSeq.
	key := "sess_x|conv_y"
	if seqCache.sessionSeqs[key] != 3 {
		t.Errorf("session seq = %d; want 3", seqCache.sessionSeqs[key])
	}
}

func TestSync_ClampLimit(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	// Insert 60 messages.
	insertSyncMessages(store, "conv_z", 1, 60)

	// Request limit=0 (invalid → clamped to 50).
	req := protocol.SyncReqPayload{
		ConvID:      "conv_z",
		LastConvSeq: 0,
		Limit:       0,
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}

	// Should return at most 50 messages (limit clamped from 0→50).
	// Since fetched limit+1=51 > 50, and 60>50, we should have 50 messages + hasMore.
	if len(res.Messages) != 50 {
		t.Errorf("len(Messages) = %d; want 50 (clamped limit)", len(res.Messages))
	}
	if !res.HasMore {
		t.Errorf("HasMore = false; want true (there are 60 msgs, limit=50)")
	}
}

func TestSync_HasMore(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	// Insert 51 messages — one more than the default limit+1 request.
	insertSyncMessages(store, "conv_w", 1, 51)

	req := protocol.SyncReqPayload{
		ConvID:      "conv_w",
		LastConvSeq: 0,
		Limit:       0, // clamped to 50, requests 51, has 51 → hasMore=true
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if len(res.Messages) != 50 {
		t.Errorf("len(Messages) = %d; want 50", len(res.Messages))
	}
	if !res.HasMore {
		t.Errorf("HasMore = false; want true")
	}
}

func TestSync_Limit100(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	insertSyncMessages(store, "conv_v", 1, 150)

	// Limit=100 is valid (not > 100), so it stays.
	req := protocol.SyncReqPayload{
		ConvID:      "conv_v",
		LastConvSeq: 0,
		Limit:       100,
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	// limit=100, fetched limit+1=101, have 150 > 101 → hasMore, return 100.
	if len(res.Messages) != 100 {
		t.Errorf("len(Messages) = %d; want 100", len(res.Messages))
	}
	if !res.HasMore {
		t.Errorf("HasMore = false; want true")
	}
}

func TestSync_Empty(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	// No messages at all.
	req := protocol.SyncReqPayload{
		ConvID:      "conv_empty",
		LastConvSeq: 0,
		Limit:       10,
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	if res.ConvID != "conv_empty" {
		t.Errorf("res.ConvID = %q; want conv_empty", res.ConvID)
	}
	if len(res.Messages) != 0 {
		t.Errorf("len(Messages) = %d; want 0", len(res.Messages))
	}
	if res.HasMore {
		t.Errorf("HasMore = true; want false")
	}
}

func TestSync_NegativeLimit(t *testing.T) {
	store := &mockMessageStore{}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	insertSyncMessages(store, "conv_n", 1, 10)

	// Negative limit → clamped to 50.
	req := protocol.SyncReqPayload{
		ConvID:      "conv_n",
		LastConvSeq: 0,
		Limit:       -5,
	}
	res, err := handler.Handle(context.Background(), "sess_x", req)
	if err != nil {
		t.Fatalf("Handle error: %v", err)
	}
	// Only 10 messages exist, so all returned, no hasMore.
	if len(res.Messages) != 10 {
		t.Errorf("len(Messages) = %d; want 10", len(res.Messages))
	}
	if res.HasMore {
		t.Errorf("HasMore = true; want false")
	}
}

func TestSync_RepoError(t *testing.T) {
	store := &mockMessageStore{syncErr: errors.New("db error")}
	seqCache := &mockSeqCache{}
	handler := NewSyncHandler(store, seqCache)

	req := protocol.SyncReqPayload{
		ConvID:      "conv_err",
		LastConvSeq: 0,
		Limit:       10,
	}
	_, err := handler.Handle(context.Background(), "sess_x", req)
	if err == nil {
		t.Fatal("expected error from repo, got nil")
	}
}

// insertSyncMessages inserts n messages into store for the given convID,
// with ConvSeq starting at firstSeq and incrementing.
func insertSyncMessages(store *mockMessageStore, convID string, firstSeq, n int) {
	for i := range n {
		seq := int64(firstSeq + i)
		msg := &model.Message{
			MsgID:   seq + 10000,
			ConvID:  convID,
			ConvSeq: seq,
			Body:    fmt.Sprintf("msg %d", seq),
		}
		store.mu.Lock()
		if store.messages == nil {
			store.messages = make(map[int64]*model.Message)
		}
		store.messages[msg.MsgID] = msg
		store.allMsgs = append(store.allMsgs, msg)
		store.mu.Unlock()
	}
}
