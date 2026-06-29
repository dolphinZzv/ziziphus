package message

import (
	"context"
	"encoding/json"
	"testing"

	pgx "github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/model"
	"siciv.space/agent/panda_ai/pkg/protocol"
)

// mockContactRequestDBWithData is a configurable mock for contact request operations.
type mockContactRequestDBWithData struct {
	requestsByFormMsgID map[int64]*model.ContactRequest
	requestsByID        map[int64]*model.ContactRequest
	updateStatusErr     error
	updateStatusCalls   []int64
}

func (m *mockContactRequestDBWithData) GetByFormMsgID(_ context.Context, formMsgID int64) (*model.ContactRequest, error) {
	if m.requestsByFormMsgID == nil {
		return nil, nil
	}
	return m.requestsByFormMsgID[formMsgID], nil
}

func (m *mockContactRequestDBWithData) GetByID(_ context.Context, id int64) (*model.ContactRequest, error) {
	if m.requestsByID == nil {
		return nil, nil
	}
	return m.requestsByID[id], nil
}

func (m *mockContactRequestDBWithData) Insert(_ context.Context, _ *model.ContactRequest) (int64, error) {
	return 1, nil
}

func (m *mockContactRequestDBWithData) UpdateStatus(_ context.Context, id int64, _ model.ContactRequestStatus) error {
	m.updateStatusCalls = append(m.updateStatusCalls, id)
	return m.updateStatusErr
}

func (m *mockContactRequestDBWithData) UpdateStatusTx(_ context.Context, _ pgx.Tx, _ int64, _ model.ContactRequestStatus) error {
	return nil
}

func (m *mockContactRequestDBWithData) LockByIDTx(_ context.Context, _ pgx.Tx, _ int64) (*model.ContactRequest, error) {
	return nil, nil
}

func (m *mockContactRequestDBWithData) UpdateFormMsgID(_ context.Context, _, _ int64) error {
	return nil
}

func (m *mockContactRequestDBWithData) Delete(_ context.Context, _ int64) error {
	return nil
}

func (m *mockContactRequestDBWithData) GetByPair(_ context.Context, _, _ string) (*model.ContactRequest, error) {
	return nil, nil
}

func (m *mockContactRequestDBWithData) ListSent(_ context.Context, _ string, _, _ int) ([]*model.ContactRequest, error) {
	return nil, nil
}

func (m *mockContactRequestDBWithData) ListReceived(_ context.Context, _ string, _, _ int) ([]*model.ContactRequest, error) {
	return nil, nil
}

func (m *mockContactRequestDBWithData) ExistsAnyDirection(_ context.Context, _, _ string) (bool, error) {
	return false, nil
}

// newIngestFixtureWithContactReq is like newIngestFixture but accepts a custom contactRequestDB.
func newIngestFixtureWithContactReq(ratePerSec, burst, maxBody int, defaultID int64, contactReqDB contactRequestDB) *Ingest {
	store := &mockMessageStore{}
	idGen := &mockIDGenerator{nextID: defaultID}
	seqCache := &mockSeqCache{}
	convMgr := &mockConvManager{}
	sessGtr := &mockSessionGetter{}
	connReg := &mockConnRegistry{}
	receiptW := &mockReceiptWriter{}
	rateLmt := NewRateLimiter(ratePerSec, burst, maxBody)
	router := NewRouter(sessGtr, convMgr, connReg)
	pusher := NewPusher(connReg, receiptW)
	return NewIngest(store, router, pusher, rateLmt, idGen, seqCache, convMgr, contactReqDB, &mockContactCreator{}, &mockUserGetter{})
}

// ---------------------------------------------------------------------------
// SendFormMessage
// ---------------------------------------------------------------------------

func TestSendFormMessage_Success(t *testing.T) {
	ing, store, idGen, seqCache, convMgr, sessGtr, connReg, _, _ := newIngestFixture(100, 100, 100000, 100)
	ctx := context.Background()

	convMgr.convs = map[string]*model.Conversation{
		"conv_form": {ConvID: "conv_form", Type: model.ConvP2P},
	}
	convMgr.members = map[string][]*model.ConvMember{
		"conv_form": {
			{ConvID: "conv_form", UserID: "user_a"},
			{ConvID: "conv_form", UserID: "user_b"},
		},
	}
	sessGtr.sessions = map[string][]string{"user_b": {"sess_b1"}}
	conn := &mockConn{}
	connReg.bySessionID = map[string]any{"sess_b1": conn}

	body := &model.FormDefinitionBody{
		Title:      "Test Form",
		FromUserID: "user_a",
	}
	msg, err := ing.SendFormMessage(ctx, "conv_form", body)
	if err != nil {
		t.Fatalf("SendFormMessage failed: %v", err)
	}

	if msg.MsgID != 100 {
		t.Errorf("msg.MsgID = %d; want 100", msg.MsgID)
	}
	if msg.ContentType != model.ContentForm {
		t.Errorf("msg.ContentType = %d; want %d", msg.ContentType, model.ContentForm)
	}
	if msg.SenderID != "" {
		t.Errorf("msg.SenderID = %q; want empty string", msg.SenderID)
	}
	if msg.Body == "" {
		t.Error("msg.Body should contain JSON")
	}

	var decoded model.FormDefinitionBody
	if err := json.Unmarshal([]byte(msg.Body), &decoded); err != nil {
		t.Fatalf("unmarshal stored body: %v", err)
	}
	if decoded.Title != "Test Form" {
		t.Errorf("decoded Title = %q; want Test Form", decoded.Title)
	}

	// Verify persisted
	if store.insertCalls != 1 {
		t.Errorf("store.Insert calls = %d; want 1", store.insertCalls)
	}
	if idGen.nextID != 101 {
		t.Errorf("idGen.nextID = %d; want 101", idGen.nextID)
	}
	if seqCache.convSeqs["conv_form"] != 1 {
		t.Errorf("conv seq = %d; want 1", seqCache.convSeqs["conv_form"])
	}

	// Verify pushed
	frames := conn.getFrames()
	if len(frames) != 1 {
		t.Fatalf("pusher sent %d frames; want 1", len(frames))
	}
}

func TestSendFormMessage_InsertError(t *testing.T) {
	ing, store, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)
	store.insertErr = errDBDown
	ctx := context.Background()

	_, err := ing.SendFormMessage(ctx, "conv_form", &model.FormDefinitionBody{Title: "test"})
	if err == nil {
		t.Fatal("expected error from failed insert")
	}
}

var errDBDown = &testError{"db down"}

type testError struct{ msg string }

func (e *testError) Error() string { return e.msg }

// ---------------------------------------------------------------------------
// ensureContacts
// ---------------------------------------------------------------------------

func TestEnsureContacts_Idempotent(t *testing.T) {
	ing, _, _, _, convMgr, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)
	ctx := context.Background()

	ing.ensureContacts(ctx, "user_a", "user_b")

	// Should have created P2P conversation
	convID := model.MakeP2PConvID("user_a", "user_b")
	conv, err := convMgr.Get(ctx, convID)
	if err != nil {
		t.Fatalf("expected P2P conv to exist after ensureContacts: %v", err)
	}
	if conv.Type != model.ConvP2P {
		t.Errorf("conv.Type = %d; want P2P", conv.Type)
	}
}

// ---------------------------------------------------------------------------
// handleFormResponse
// ---------------------------------------------------------------------------

func TestHandleFormResponse_InvalidBody(t *testing.T) {
	ing, _, _, _, _, _, _, _, _ := newIngestFixture(100, 100, 100000, 100)
	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_b"),
		ContentType: int(model.ContentFormResponse),
		Body:        "not-json",
		ClientSeq:   1,
	}

	_, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err == nil {
		t.Fatal("expected error for invalid JSON body")
	}
}

func TestHandleFormResponse_Dedup(t *testing.T) {
	crdb := &mockContactRequestDBWithData{}
	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	// Insert a message that already exists for this client_seq
	ing.store.(*mockMessageStore).Insert(ctx, &model.Message{
		MsgID:           999,
		ConvID:          model.MakeSystemConvID("user_b"),
		SenderID:        "user_b",
		SenderSessionID: "sess_b1",
		ClientSeq:       5,
		Timestamp:       100000,
		Status:          model.MsgSent,
	})

	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_b"),
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve"}`,
		ClientSeq:   5,
	}

	ack, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if ack.MsgID != 999 {
		t.Errorf("ack.MsgID = %d; want 999 (existing)", ack.MsgID)
	}
}

func TestHandleFormResponse_ContactRequestNotFound(t *testing.T) {
	crdb := &mockContactRequestDBWithData{}

	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_b"),
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":999,"request_id":999,"action":"approve"}`,
		ClientSeq:   10,
	}

	_, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err == nil {
		t.Fatal("expected error for non-existent contact request")
	}
}

func TestHandleFormResponse_WrongSender(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestPending},
		},
	}
	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	// user_c is not the target (ToUserID=user_b), should fail
	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_c"),
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve"}`,
		ClientSeq:   10,
	}

	_, err := ing.Ingest(ctx, "user_c", "sess_c1", payload)
	if err == nil {
		t.Fatal("expected error for wrong sender")
	}
}

func TestHandleFormResponse_WrongConv(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestPending},
		},
	}
	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      "wrong_conv",
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve"}`,
		ClientSeq:   10,
	}

	_, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err == nil {
		t.Fatal("expected error for wrong conversation")
	}
}

func TestHandleFormResponse_InvalidAction(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestPending},
		},
	}
	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_b"),
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"invalid"}`,
		ClientSeq:   10,
	}

	_, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
}

func TestHandleFormResponse_ApproveSuccess(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestPending},
		},
	}
	store := &mockMessageStore{}
	idGen := &mockIDGenerator{nextID: 100}
	seqCache := &mockSeqCache{}
	convMgr := &mockConvManager{
		convs: map[string]*model.Conversation{
			"sys:user_b": {ConvID: "sys:user_b", Type: model.ConvSystem, OwnerID: "user_b"},
		},
	}
	sessGtr := &mockSessionGetter{}
	connReg := &mockConnRegistry{}
	rateLmt := NewRateLimiter(100, 100, 100000)
	router := NewRouter(sessGtr, convMgr, connReg)
	pusher := NewPusher(connReg, &mockReceiptWriter{})
	ing := NewIngest(store, router, pusher, rateLmt, idGen, seqCache, convMgr, crdb, &mockContactCreator{}, &mockUserGetter{})

	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      "sys:user_b",
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve","responder_name":"Bob"}`,
		ClientSeq:   10,
	}

	ack, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if ack.MsgID != 100 {
		t.Errorf("ack.MsgID = %d; want 100", ack.MsgID)
	}
	if ack.Status != int(model.MsgSent) {
		t.Errorf("ack.Status = %d; want MsgSent", ack.Status)
	}

	// Verify contact request status was updated
	if len(crdb.updateStatusCalls) != 1 {
		t.Errorf("UpdateStatus calls = %d; want 1", len(crdb.updateStatusCalls))
	}
}

func TestHandleFormResponse_AlreadyApproved_Idempotent(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestApproved},
		},
	}
	store := &mockMessageStore{}
	idGen := &mockIDGenerator{nextID: 100}
	seqCache := &mockSeqCache{}
	convMgr := &mockConvManager{
		convs: map[string]*model.Conversation{
			"sys:user_b": {ConvID: "sys:user_b", Type: model.ConvSystem, OwnerID: "user_b"},
		},
	}
	sessGtr := &mockSessionGetter{}
	connReg := &mockConnRegistry{}
	rateLmt := NewRateLimiter(100, 100, 100000)
	router := NewRouter(sessGtr, convMgr, connReg)
	pusher := NewPusher(connReg, &mockReceiptWriter{})
	ing := NewIngest(store, router, pusher, rateLmt, idGen, seqCache, convMgr, crdb, &mockContactCreator{}, &mockUserGetter{})

	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      "sys:user_b",
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve"}`,
		ClientSeq:   10,
	}

	ack, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err != nil {
		t.Fatalf("Ingest failed for already-approved request: %v", err)
	}
	if ack.Status != int(model.MsgSent) {
		t.Errorf("ack.Status = %d; want MsgSent", ack.Status)
	}
}

func TestHandleFormResponse_RejectSuccess(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestPending},
		},
	}
	store := &mockMessageStore{}
	idGen := &mockIDGenerator{nextID: 100}
	seqCache := &mockSeqCache{}
	convMgr := &mockConvManager{
		convs: map[string]*model.Conversation{
			"sys:user_b": {ConvID: "sys:user_b", Type: model.ConvSystem, OwnerID: "user_b"},
		},
	}
	sessGtr := &mockSessionGetter{}
	connReg := &mockConnRegistry{}
	rateLmt := NewRateLimiter(100, 100, 100000)
	router := NewRouter(sessGtr, convMgr, connReg)
	pusher := NewPusher(connReg, &mockReceiptWriter{})
	ing := NewIngest(store, router, pusher, rateLmt, idGen, seqCache, convMgr, crdb, &mockContactCreator{}, &mockUserGetter{})

	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      "sys:user_b",
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"reject","responder_name":"Bob"}`,
		ClientSeq:   10,
	}

	ack, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err != nil {
		t.Fatalf("Ingest failed: %v", err)
	}
	if ack.MsgID != 100 {
		t.Errorf("ack.MsgID = %d; want 100", ack.MsgID)
	}
	if len(crdb.updateStatusCalls) != 1 {
		t.Errorf("UpdateStatus calls = %d; want 1", len(crdb.updateStatusCalls))
	}
}

func TestHandleFormResponse_AlreadyHandled_Error(t *testing.T) {
	crdb := &mockContactRequestDBWithData{
		requestsByFormMsgID: map[int64]*model.ContactRequest{
			1: {ID: 1, FromUserID: "user_a", ToUserID: "user_b", Status: model.ContactRequestRejected},
		},
	}
	ing := newIngestFixtureWithContactReq(100, 100, 100000, 100, crdb)
	ctx := context.Background()

	payload := protocol.MsgSendPayload{
		ConvID:      model.MakeSystemConvID("user_b"),
		ContentType: int(model.ContentFormResponse),
		Body:        `{"form_msg_id":1,"request_id":1,"action":"approve"}`,
		ClientSeq:   10,
	}

	_, err := ing.Ingest(ctx, "user_b", "sess_b1", payload)
	if err == nil {
		t.Fatal("expected error for already-handled request")
	}
}
