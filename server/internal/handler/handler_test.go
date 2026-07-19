package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"ziziphus/internal/auth"
	"ziziphus/internal/gateway"
	"ziziphus/pkg/model"
	"ziziphus/pkg/protocol"
)

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

type mockSessionManager struct {
	createFunc     func(ctx context.Context, userID string, device model.DeviceType, deviceName string, clientIP string, deviceID string) (*model.Session, error)
	getFunc        func(ctx context.Context, sessionID string) *model.Session
	deleteFunc     func(ctx context.Context, sessionID string) error
	bindFunc       func(ctx context.Context, sessionID, connID string) error
	mu             sync.Mutex
	deleteCalled   bool
	bindCalled     bool
	lastBindArgs   [2]string // sessionID, connID
	lastCreateUser string
	lastCreateDev  model.DeviceType
}

func (m *mockSessionManager) GetUserSessionIDs(ctx context.Context, userID string) []string {
	return nil
}

func (m *mockSessionManager) Create(ctx context.Context, userID string, device model.DeviceType, deviceName string, clientIP string, deviceID string) (*model.Session, error) {
	m.mu.Lock()
	m.lastCreateUser = userID
	m.lastCreateDev = device
	m.mu.Unlock()
	if m.createFunc != nil {
		return m.createFunc(ctx, userID, device, deviceName, clientIP, deviceID)
	}
	return &model.Session{SessionID: "sess1", UserID: userID}, nil
}

func (m *mockSessionManager) Get(ctx context.Context, sessionID string) *model.Session {
	if m.getFunc != nil {
		return m.getFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionManager) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	m.deleteCalled = true
	m.mu.Unlock()
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionManager) BindConnection(ctx context.Context, sessionID, connID string) error {
	m.mu.Lock()
	m.bindCalled = true
	m.lastBindArgs = [2]string{sessionID, connID}
	m.mu.Unlock()
	if m.bindFunc != nil {
		return m.bindFunc(ctx, sessionID, connID)
	}
	return nil
}

func (m *mockSessionManager) wasDeleteCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.deleteCalled
}

func (m *mockSessionManager) wasBindCalled() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.bindCalled
}

func (m *mockSessionManager) getLastBindArgs() (string, string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.lastBindArgs[0], m.lastBindArgs[1]
}

type mockMessageIngester struct {
	ingestFunc func(ctx context.Context, senderID, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error)
	calls      int
	mu         sync.Mutex
	lastSender string
}

func (m *mockMessageIngester) Ingest(ctx context.Context, senderID, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
	m.mu.Lock()
	m.calls++
	m.lastSender = senderID
	m.mu.Unlock()
	if m.ingestFunc != nil {
		return m.ingestFunc(ctx, senderID, sessionID, payload)
	}
	return nil, errors.New("ingest not mocked")
}

type mockSyncHandler struct {
	handleFunc func(ctx context.Context, sessionID string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error)
	calls      int
	mu         sync.Mutex
}

func (m *mockSyncHandler) Handle(ctx context.Context, sessionID string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if m.handleFunc != nil {
		return m.handleFunc(ctx, sessionID, req)
	}
	return nil, errors.New("sync not mocked")
}

type mockReadReceiptHandler struct {
	markReadFunc func(ctx context.Context, userID, convID string, msgID int64) error
	calls        int
	mu           sync.Mutex
	lastUserID   string
	lastConvID   string
	lastMsgID    int64
}

func (m *mockReadReceiptHandler) MarkRead(ctx context.Context, userID, convID string, msgID int64) error {
	m.mu.Lock()
	m.calls++
	m.lastUserID = userID
	m.lastConvID = convID
	m.lastMsgID = msgID
	m.mu.Unlock()
	if m.markReadFunc != nil {
		return m.markReadFunc(ctx, userID, convID, msgID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: msgEditor
// ---------------------------------------------------------------------------

type mockMsgEditor struct {
	getFunc        func(ctx context.Context, msgID int64) (*model.Message, error)
	updateBodyFunc func(ctx context.Context, msgID int64, newBody string) error
	recallFunc     func(ctx context.Context, msgID int64) error
}

func (m *mockMsgEditor) Get(ctx context.Context, msgID int64) (*model.Message, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, msgID)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockMsgEditor) UpdateBody(ctx context.Context, msgID int64, newBody string) error {
	if m.updateBodyFunc != nil {
		return m.updateBodyFunc(ctx, msgID, newBody)
	}
	return nil
}

func (m *mockMsgEditor) Recall(ctx context.Context, msgID int64) error {
	if m.recallFunc != nil {
		return m.recallFunc(ctx, msgID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// setupConnPair creates a WebSocket server and client pair connected to each
// other, returning the server-side connection (for wrapping in a
// *gateway.Connection) and the client-side connection (for reading responses).
func setupConnPair(t *testing.T) (serverConn *websocket.Conn, clientConn *websocket.Conn) {
	t.Helper()

	connCh := make(chan *websocket.Conn, 1)
	doneCh := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := up.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("server upgrade failed: %v", err)
			return
		}
		connCh <- conn
		// Block to keep the handler alive; the connection remains open and
		// usable for reading/writing until doneCh is closed.
		<-doneCh
	}))

	url := "ws" + strings.TrimPrefix(srv.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("client dial failed: %v", err)
	}

	t.Cleanup(func() {
		client.Close()
		close(doneCh)
		srv.Close()
	})

	serverConn = <-connCh
	return serverConn, client
}

// newHandler returns a WSHandler wired with sensible defaults. Each call
// creates a brand-new gateway.Manager so tests are isolated.
func newHandler(
	authMW func(ctx context.Context, token string) (context.Context, error),
	sessMgr sessionManager,
	ingest messageIngester,
	sync syncHandler,
	receipt readReceiptHandler,
	msgRepo ...msgEditor,
) *WSHandler {
	mr := msgEditor(&mockMsgEditor{})
	if len(msgRepo) > 0 {
		mr = msgRepo[0]
	}
	return NewWSHandler(
		authMW,
		sessMgr,
		gateway.NewManager(),
		ingest,
		sync,
		receipt,
		mr,
	)
}

// defaultHandler builds a WSHandler whose mocks all return success / no-op.
// It creates a fresh gateway.Manager each call.
func defaultHandler() *WSHandler {
	return newHandler(
		func(ctx context.Context, token string) (context.Context, error) {
			return context.WithValue(ctx, auth.CtxKeyUserID, "user1"), nil
		},
		&mockSessionManager{},
		&mockMessageIngester{
			ingestFunc: func(_ context.Context, _, _ string, _ protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
				return &protocol.MsgSendAckPayload{MsgID: 100, Timestamp: time.Now().UnixMilli(), ClientSeq: 1, Status: 0}, nil
			},
		},
		&mockSyncHandler{
			handleFunc: func(_ context.Context, _ string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
				return &protocol.SyncResPayload{ConvID: req.ConvID, Messages: []protocol.SyncMessage{}, HasMore: false}, nil
			},
		},
		&mockReadReceiptHandler{},
	)
}

// readFrame is a small helper that reads a single protocol.Frame from conn
// with a short read deadline.
func readFrame(t *testing.T, conn *websocket.Conn) protocol.Frame {
	t.Helper()
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var f protocol.Frame
	if err := conn.ReadJSON(&f); err != nil {
		t.Fatalf("read frame failed: %v", err)
	}
	return f
}

// ---------------------------------------------------------------------------
// 1) Ping / Pong
// ---------------------------------------------------------------------------

func TestDispatch_Ping(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.Ping, ID: "ping-1"}, gwConn)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.Pong {
		t.Fatalf("expected Pong (%d), got Type=%d ID=%s", protocol.Pong, resp.Type, resp.ID)
	}
}

// ---------------------------------------------------------------------------
// 2) MsgSend success
// ---------------------------------------------------------------------------

func TestDispatch_MsgSend_Success(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	ingest := &mockMessageIngester{
		ingestFunc: func(_ context.Context, senderID, sessionID string, payload protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
			if senderID != "user1" {
				t.Errorf("expected senderID user1, got %s", senderID)
			}
			if sessionID != "sess1" {
				t.Errorf("expected sessionID sess1, got %s", sessionID)
			}
			if payload.ConvID != "conv-a" {
				t.Errorf("expected convID conv-a, got %s", payload.ConvID)
			}
			return &protocol.MsgSendAckPayload{MsgID: 42, Timestamp: 1_700_000_000_000, ClientSeq: payload.ClientSeq, Status: 0}, nil
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		ingest,
		defaultHandler().sync,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.MsgSendPayload{
		ConvID: "conv-a", ContentType: 1, Body: "hello world", ClientSeq: 99,
	})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.MsgSend, ID: "msg-1", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.MsgSendAck {
		t.Fatalf("expected MsgSendAck (%d), got Type=%d", protocol.MsgSendAck, resp.Type)
	}
	if resp.ID != "msg-1" {
		t.Fatalf("expected ID msg-1, got %s", resp.ID)
	}

	var ack protocol.MsgSendAckPayload
	if err := json.Unmarshal(resp.Payload, &ack); err != nil {
		t.Fatalf("unmarshal MsgSendAckPayload: %v", err)
	}
	if ack.MsgID != 42 {
		t.Fatalf("expected MsgID 42, got %d", ack.MsgID)
	}
	if ack.ClientSeq != 99 {
		t.Fatalf("expected ClientSeq 99, got %d", ack.ClientSeq)
	}
}

// ---------------------------------------------------------------------------
// 3) MsgSend error – AppError
// ---------------------------------------------------------------------------

func TestDispatch_MsgSend_AppError(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	ingest := &mockMessageIngester{
		ingestFunc: func(_ context.Context, _, _ string, _ protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
			return nil, model.ErrNotInConv
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		ingest,
		defaultHandler().sync,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.MsgSendPayload{ConvID: "conv-a", Body: "x"})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.MsgSend, ID: "err-1", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch should return nil after sending error frame, got: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.Error {
		t.Fatalf("expected Error (%d), got Type=%d", protocol.Error, resp.Type)
	}
	if resp.ID != "err-1" {
		t.Fatalf("expected ID err-1, got %s", resp.ID)
	}
	var errPayload protocol.ErrorPayload
	if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal ErrorPayload: %v", err)
	}
	if errPayload.Code != model.ErrNotInConv.Code {
		t.Fatalf("expected err code %d, got %d", model.ErrNotInConv.Code, errPayload.Code)
	}
}

// ---------------------------------------------------------------------------
// 3b) MsgSend error – generic (non-AppError) error
// ---------------------------------------------------------------------------

func TestDispatch_MsgSend_GenericError(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	ingest := &mockMessageIngester{
		ingestFunc: func(_ context.Context, _, _ string, _ protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
			return nil, errors.New("something went wrong")
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		ingest,
		defaultHandler().sync,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.MsgSendPayload{ConvID: "conv-a", Body: "x"})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.MsgSend, ID: "err-2", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch should return nil after sending error frame, got: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.Error {
		t.Fatalf("expected Error (%d), got Type=%d", protocol.Error, resp.Type)
	}
	var errPayload protocol.ErrorPayload
	if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal ErrorPayload: %v", err)
	}
	// generic error → ErrInternal
	if errPayload.Code != model.ErrInternal {
		t.Fatalf("expected ErrInternal (%d), got %d", model.ErrInternal, errPayload.Code)
	}
}

// ---------------------------------------------------------------------------
// 3c) MsgSend invalid payload
// ---------------------------------------------------------------------------

func TestDispatch_MsgSend_InvalidPayload(t *testing.T) {
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{
		Type:    protocol.MsgSend,
		ID:      "bad",
		Payload: json.RawMessage(`{bad json}`),
	}, gwConn)
	if err == nil {
		t.Fatal("expected error for invalid MsgSend payload, got nil")
	}
}

// ---------------------------------------------------------------------------
// 4) SyncReq success
// ---------------------------------------------------------------------------

func TestDispatch_SyncReq_Success(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	syncH := &mockSyncHandler{
		handleFunc: func(_ context.Context, sessionID string, req protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
			if sessionID != "sess1" {
				t.Errorf("expected sessionID sess1, got %s", sessionID)
			}
			return &protocol.SyncResPayload{
				ConvID:   req.ConvID,
				Messages: []protocol.SyncMessage{{MsgID: 10, Body: "hi"}},
				HasMore:  false,
			}, nil
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		syncH,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.SyncReqPayload{ConvID: "conv-x", LastConvSeq: 5, Limit: 20})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.SyncReq, ID: "sync-1", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.SyncRes {
		t.Fatalf("expected SyncRes (%d), got Type=%d", protocol.SyncRes, resp.Type)
	}
	if resp.ID != "sync-1" {
		t.Fatalf("expected ID sync-1, got %s", resp.ID)
	}

	var res protocol.SyncResPayload
	if err := json.Unmarshal(resp.Payload, &res); err != nil {
		t.Fatalf("unmarshal SyncResPayload: %v", err)
	}
	if res.ConvID != "conv-x" {
		t.Fatalf("expected convID conv-x, got %s", res.ConvID)
	}
	if len(res.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(res.Messages))
	}
}

// ---------------------------------------------------------------------------
// 5) SyncReq error – AppError
// ---------------------------------------------------------------------------

func TestDispatch_SyncReq_AppError(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	syncH := &mockSyncHandler{
		handleFunc: func(_ context.Context, _ string, _ protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
			return nil, model.NewAppError(model.ErrNotFound, "not found")
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		syncH,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.SyncReqPayload{ConvID: "conv-x"})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.SyncReq, ID: "err-sync", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch should return nil after sending error frame, got: %v", err)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.Error {
		t.Fatalf("expected Error (%d), got Type=%d", protocol.Error, resp.Type)
	}
	var errPayload protocol.ErrorPayload
	if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal ErrorPayload: %v", err)
	}
	if errPayload.Code != model.ErrNotFound {
		t.Fatalf("expected ErrNotFound (%d), got %d", model.ErrNotFound, errPayload.Code)
	}
}

// ---------------------------------------------------------------------------
// 5b) SyncReq error – generic error
// ---------------------------------------------------------------------------

func TestDispatch_SyncReq_GenericError(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	syncH := &mockSyncHandler{
		handleFunc: func(_ context.Context, _ string, _ protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
			return nil, errors.New("sync boom")
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		syncH,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.SyncReqPayload{ConvID: "conv-x"})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.SyncReq, ID: "err-sync2", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch should return nil after sending error frame, got: %v", err)
	}

	resp := readFrame(t, clientConn)
	var errPayload protocol.ErrorPayload
	if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal ErrorPayload: %v", err)
	}
	if errPayload.Code != model.ErrInternal {
		t.Fatalf("expected ErrInternal (%d), got %d", model.ErrInternal, errPayload.Code)
	}
}

// ---------------------------------------------------------------------------
// 5c) SyncReq invalid payload
// ---------------------------------------------------------------------------

func TestDispatch_SyncReq_InvalidPayload(t *testing.T) {
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{
		Type:    protocol.SyncReq,
		ID:      "bad",
		Payload: json.RawMessage(`not-json`),
	}, gwConn)
	if err == nil {
		t.Fatal("expected error for invalid SyncReq payload, got nil")
	}
}

// ---------------------------------------------------------------------------
// 6) MsgReadNotify
// ---------------------------------------------------------------------------

func TestDispatch_MsgReadNotify_Success(t *testing.T) {
	receipt := &mockReadReceiptHandler{}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		defaultHandler().sync,
		receipt,
	)
	// No WS connection needed – dispatch does not send a frame for MsgReadNotify.
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	payload, _ := json.Marshal(protocol.MsgReadNotifyPayload{
		ConvID: "conv-r", UserID: "user1", MsgID: 77,
	})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.MsgReadNotify, Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	if receipt.calls != 1 {
		t.Fatalf("expected 1 MarkRead call, got %d", receipt.calls)
	}
	if receipt.lastUserID != "user1" {
		t.Fatalf("expected userID user1, got %s", receipt.lastUserID)
	}
	if receipt.lastConvID != "conv-r" {
		t.Fatalf("expected convID conv-r, got %s", receipt.lastConvID)
	}
	if receipt.lastMsgID != 77 {
		t.Fatalf("expected msgID 77, got %d", receipt.lastMsgID)
	}
}

func TestDispatch_MsgReadNotify_Error(t *testing.T) {
	receipt := &mockReadReceiptHandler{
		markReadFunc: func(_ context.Context, _, _ string, _ int64) error {
			return errors.New("mark read failed")
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		defaultHandler().sync,
		receipt,
	)
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	payload, _ := json.Marshal(protocol.MsgReadNotifyPayload{
		ConvID: "conv-r", UserID: "user1", MsgID: 99,
	})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.MsgReadNotify, Payload: payload}, gwConn)
	if err == nil {
		t.Fatal("expected error from MarkRead, got nil")
	}
}

func TestDispatch_MsgReadNotify_InvalidPayload(t *testing.T) {
	h := newHandler(
		defaultHandler().authMW,
		&mockSessionManager{},
		defaultHandler().ingest,
		defaultHandler().sync,
		&mockReadReceiptHandler{},
	)

	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{
		Type:    protocol.MsgReadNotify,
		Payload: json.RawMessage(`{{bad}}`),
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

// ---------------------------------------------------------------------------
// 7) SessionRecover
// ---------------------------------------------------------------------------

func TestDispatch_SessionRecover(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	sessMgr := &mockSessionManager{
		getFunc: func(_ context.Context, sessionID string) *model.Session {
			return &model.Session{SessionID: sessionID, UserID: "user1"}
		},
	}
	h := newHandler(
		defaultHandler().authMW,
		sessMgr,
		defaultHandler().ingest,
		defaultHandler().sync,
		defaultHandler().receipt,
	)
	gwConn := gateway.NewConnection("conn-recover", "user1", "", int(model.DeviceDesktop), serverConn)

	payload, _ := json.Marshal(protocol.SessionRecoverPayload{SessionID: "sess-recover"})
	err := h.dispatch(context.Background(), "user1", "", protocol.Frame{Type: protocol.SessionRecover, ID: "rec-1", Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("dispatch returned error: %v", err)
	}

	// Verify BindConnection was called on the recovered session.
	sessID, connID := sessMgr.getLastBindArgs()
	if sessID != "sess-recover" {
		t.Fatalf("expected sessionID sess-recover, got %s", sessID)
	}
	if connID != "conn-recover" {
		t.Fatalf("expected connID conn-recover, got %s", connID)
	}

	resp := readFrame(t, clientConn)
	if resp.Type != protocol.SessionRecoverAck {
		t.Fatalf("expected SessionRecoverAck (%d), got Type=%d", protocol.SessionRecoverAck, resp.Type)
	}
	if resp.ID != "rec-1" {
		t.Fatalf("expected ID rec-1, got %s", resp.ID)
	}

	var ack protocol.SessionRecoverAckPayload
	if err := json.Unmarshal(resp.Payload, &ack); err != nil {
		t.Fatalf("unmarshal SessionRecoverAckPayload: %v", err)
	}
	if ack.SessionID != "sess-recover" {
		t.Fatalf("expected SessionID sess-recover, got %s", ack.SessionID)
	}
	if ack.UserID != "user1" {
		t.Fatalf("expected UserID user1, got %s", ack.UserID)
	}
	if ack.Timestamp == 0 {
		t.Fatal("expected non-zero timestamp")
	}
}

func TestDispatch_SessionRecover_InvalidPayload(t *testing.T) {
	h := defaultHandler()

	err := h.dispatch(context.Background(), "user1", "", protocol.Frame{
		Type:    protocol.SessionRecover,
		Payload: json.RawMessage(`{bad}`),
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid payload, got nil")
	}
}

// ---------------------------------------------------------------------------
// 8) Unknown frame type
// ---------------------------------------------------------------------------

func TestDispatch_UnknownFrameType(t *testing.T) {
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	// Frame type 999 is not handled.
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: 999, Payload: json.RawMessage(`{"a":1}`)}, gwConn)
	if err != nil {
		t.Fatalf("unknown frame type should return nil, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// 9) Typing – no-op
// ---------------------------------------------------------------------------

func TestDispatch_Typing(t *testing.T) {
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn1", "user1", "sess1", int(model.DeviceDesktop), nil)

	payload, _ := json.Marshal(protocol.TypingPayload{
		ConvID: "conv-t", UserID: "user1", SessionID: "sess1",
	})
	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{Type: protocol.Typing, Payload: payload}, gwConn)
	if err != nil {
		t.Fatalf("typing frame should return nil, got: %v", err)
	}
}

func TestDispatch_Typing_InvalidPayload(t *testing.T) {
	h := defaultHandler()

	err := h.dispatch(context.Background(), "user1", "sess1", protocol.Frame{
		Type:    protocol.Typing,
		Payload: json.RawMessage(`bad`),
	}, nil)
	if err == nil {
		t.Fatal("expected error for invalid Typing payload, got nil")
	}
}

// ---------------------------------------------------------------------------
// 10) ServeHTTP with valid token – full flow
// ---------------------------------------------------------------------------

func TestServeHTTP_FullFlow(t *testing.T) {
	connMgr := gateway.NewManager()

	sessMgr := &mockSessionManager{
		createFunc: func(ctx context.Context, userID string, device model.DeviceType, deviceName string, clientIP string, deviceID string) (*model.Session, error) {
			if device != model.DeviceDesktop {
				t.Errorf("expected DeviceDesktop (%d), got %d", model.DeviceDesktop, device)
			}
			if deviceName != "macOS" {
				t.Errorf("expected deviceName macOS, got %s", deviceName)
			}
			return &model.Session{SessionID: "sess-flow", UserID: userID}, nil
		},
	}

	h := NewWSHandler(
		func(ctx context.Context, token string) (context.Context, error) {
			return context.WithValue(ctx, auth.CtxKeyUserID, "user-flow"), nil
		},
		sessMgr,
		connMgr,
		&mockMessageIngester{
			ingestFunc: func(_ context.Context, _, _ string, p protocol.MsgSendPayload) (*protocol.MsgSendAckPayload, error) {
				return &protocol.MsgSendAckPayload{MsgID: 200, Timestamp: time.Now().UnixMilli(), ClientSeq: p.ClientSeq, Status: 0}, nil
			},
		},
		&mockSyncHandler{
			handleFunc: func(_ context.Context, _ string, _ protocol.SyncReqPayload) (*protocol.SyncResPayload, error) {
				return &protocol.SyncResPayload{ConvID: "c1", Messages: nil, HasMore: false}, nil
			},
		},
		&mockReadReceiptHandler{},
		&mockMsgEditor{},
	)

	server := httptest.NewServer(h)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	// ---- wait for handler to finish initialisation (runs in separate goroutine) ----
	waitFor := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(waitFor) {
		if connMgr.Count() == 1 && sessMgr.wasBindCalled() {
			goto initialised
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("handler init did not complete: connMgr.Count()=%d bindCalled=%v", connMgr.Count(), sessMgr.wasBindCalled())

initialised:

	// ---- expect welcome (SessionRecoverAck) ----
	welcome := readFrame(t, conn)
	if welcome.Type != protocol.SessionRecoverAck {
		t.Fatalf("expected SessionRecoverAck (%d) welcome frame, got %d", protocol.SessionRecoverAck, welcome.Type)
	}

	// ---- send Ping, expect Pong ----
	if err := conn.WriteJSON(protocol.Frame{Type: protocol.Ping, ID: "f1"}); err != nil {
		t.Fatalf("write ping failed: %v", err)
	}
	resp := readFrame(t, conn)
	if resp.Type != protocol.Pong {
		t.Fatalf("expected Pong (%d), got %d", protocol.Pong, resp.Type)
	}

	// ---- send MsgSend, expect MsgSendAck ----
	msgPayload, _ := json.Marshal(protocol.MsgSendPayload{ConvID: "c1", Body: "hi", ClientSeq: 7})
	if err := conn.WriteJSON(protocol.Frame{Type: protocol.MsgSend, ID: "f2", Payload: msgPayload}); err != nil {
		t.Fatalf("write MsgSend failed: %v", err)
	}
	resp = readFrame(t, conn)
	if resp.Type != protocol.MsgSendAck {
		t.Fatalf("expected MsgSendAck (%d), got %d", protocol.MsgSendAck, resp.Type)
	}

	// ---- send SyncReq, expect SyncRes ----
	syncPayload, _ := json.Marshal(protocol.SyncReqPayload{ConvID: "c1", LastConvSeq: 0, Limit: 50})
	if err := conn.WriteJSON(protocol.Frame{Type: protocol.SyncReq, ID: "f3", Payload: syncPayload}); err != nil {
		t.Fatalf("write SyncReq failed: %v", err)
	}
	resp = readFrame(t, conn)
	if resp.Type != protocol.SyncRes {
		t.Fatalf("expected SyncRes (%d), got %d", protocol.SyncRes, resp.Type)
	}

	// ---- close client to trigger cleanup ----
	conn.Close()

	// ---- wait for cleanup ----
	waitFor = time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(waitFor) {
		if connMgr.Count() == 0 {
			goto cleaned
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("cleanup did not complete within timeout")

cleaned:
	// Connection count is zero.
}

// ---------------------------------------------------------------------------
// 11) ServeHTTP with invalid token
// ---------------------------------------------------------------------------

func TestServeHTTP_InvalidToken(t *testing.T) {
	h := NewWSHandler(
		func(ctx context.Context, token string) (context.Context, error) {
			return nil, errors.New("bad token")
		},
		&mockSessionManager{},
		gateway.NewManager(),
		&mockMessageIngester{},
		&mockSyncHandler{},
		&mockReadReceiptHandler{},
		&mockMsgEditor{},
	)

	server := httptest.NewServer(h)
	defer server.Close()

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial should succeed (upgrade before auth), got: %v", err)
	}
	defer conn.Close()

	resp := readFrame(t, conn)
	if resp.Type != protocol.Error {
		t.Fatalf("expected Error (%d) frame, got Type=%d", protocol.Error, resp.Type)
	}
	var errPayload protocol.ErrorPayload
	if err := json.Unmarshal(resp.Payload, &errPayload); err != nil {
		t.Fatalf("unmarshal ErrorPayload: %v", err)
	}
	if errPayload.Code != model.ErrNoPermission {
		t.Fatalf("expected ErrNoPermission (%d), got %d", model.ErrNoPermission, errPayload.Code)
	}
	if errPayload.Message == "" {
		t.Fatal("expected non-empty error message")
	}
}

// ---------------------------------------------------------------------------
// 12) readLoop with normal closure
// ---------------------------------------------------------------------------

func TestReadLoop_NormalClosure(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn-rl", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	// Close the client connection to trigger a normal WS close.
	if err := clientConn.Close(); err != nil {
		t.Fatalf("client close failed: %v", err)
	}

	// readLoop should return gracefully (no panic, no hanging).
	done := make(chan struct{})
	go func() {
		defer close(done)
		h.readLoop(context.Background(), gwConn, "user1", "sess1")
	}()

	select {
	case <-done:
		// Success – readLoop returned.
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not return within 3s after client close")
	}
}

// ---------------------------------------------------------------------------
// Edge case: readLoop returns on general read error (non-close)
// ---------------------------------------------------------------------------

func TestReadLoop_ReadError(t *testing.T) {
	serverConn, clientConn := setupConnPair(t)
	h := defaultHandler()
	gwConn := gateway.NewConnection("conn-rl2", "user1", "sess1", int(model.DeviceDesktop), serverConn)

	// Send a malformed message to trigger a read error on the server.
	// A close with abnormal code should cause ReadFrame to return an error
	// that is not CloseNormalClosure / CloseGoingAway.
	if err := clientConn.WriteMessage(websocket.CloseMessage,
		websocket.FormatCloseMessage(websocket.CloseProtocolError, "")); err != nil {
		t.Fatalf("write close frame failed: %v", err)
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		h.readLoop(context.Background(), gwConn, "user1", "sess1")
	}()

	select {
	case <-done:
		// readLoop returned after logging the error.
	case <-time.After(3 * time.Second):
		t.Fatal("readLoop did not return within 3s after abnormal close")
	}
}
