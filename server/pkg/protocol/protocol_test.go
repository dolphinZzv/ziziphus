package protocol

import (
	"encoding/json"
	"testing"
)

// ---------------------------------------------------------------------------
// MessageType constants
// ---------------------------------------------------------------------------

func TestMessageType_Constants(t *testing.T) {
	tests := []struct {
		name string
		got  MessageType
		want MessageType
	}{
		{"MsgSend", MsgSend, 1},
		{"MsgSendAck", MsgSendAck, 2},
		{"MsgPush", MsgPush, 11},
		{"MsgReceived", MsgReceived, 12},
		{"SyncReq", SyncReq, 21},
		{"SyncRes", SyncRes, 22},
		{"MsgReadNotify", MsgReadNotify, 32},
		{"SessionOnline", SessionOnline, 41},
		{"SessionOffline", SessionOffline, 42},
		{"SessionRecover", SessionRecover, 43},
		{"SessionRecoverAck", SessionRecoverAck, 44},
		{"Typing", Typing, 51},
		{"Ping", Ping, 61},
		{"Pong", Pong, 62},
		{"Error", Error, 71},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("%s = %d, want %d", tt.name, tt.got, tt.want)
			}
		})
	}
}

func TestMessageType_Unique(t *testing.T) {
	m := make(map[MessageType]string)
	all := []struct {
		name string
		val  MessageType
	}{
		{"MsgSend", MsgSend},
		{"MsgSendAck", MsgSendAck},
		{"MsgPush", MsgPush},
		{"MsgReceived", MsgReceived},
		{"SyncReq", SyncReq},
		{"SyncRes", SyncRes},
		{"MsgReadNotify", MsgReadNotify},
		{"SessionOnline", SessionOnline},
		{"SessionOffline", SessionOffline},
		{"SessionRecover", SessionRecover},
		{"SessionRecoverAck", SessionRecoverAck},
		{"Typing", Typing},
		{"Ping", Ping},
		{"Pong", Pong},
		{"Error", Error},
	}
	for _, x := range all {
		if prev, ok := m[x.val]; ok {
			t.Errorf("duplicate MessageType value %d: %q and %q", x.val, prev, x.name)
		}
		m[x.val] = x.name
	}
}

// ---------------------------------------------------------------------------
// Frame – JSON round-trip
// ---------------------------------------------------------------------------

func TestFrame_JSONRoundTrip(t *testing.T) {
	raw := json.RawMessage(`{"foo":"bar"}`)
	f := Frame{
		Type:    MsgSend,
		ID:      "msg-001",
		Payload: raw,
	}

	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal Frame: %v", err)
	}

	var f2 Frame
	if err := json.Unmarshal(data, &f2); err != nil {
		t.Fatalf("Unmarshal Frame: %v", err)
	}

	if f2.Type != MsgSend {
		t.Errorf("Type = %d, want %d", f2.Type, MsgSend)
	}
	if f2.ID != "msg-001" {
		t.Errorf("ID = %q, want %q", f2.ID, "msg-001")
	}
	if string(f2.Payload) != string(raw) {
		t.Errorf("Payload = %s, want %s", string(f2.Payload), string(raw))
	}
}

func TestFrame_TypeRoundTrip(t *testing.T) {
	types := []MessageType{
		MsgSend, MsgSendAck, MsgPush, MsgReceived,
		SyncReq, SyncRes, MsgReadNotify,
		SessionOnline, SessionOffline, SessionRecover, SessionRecoverAck,
		Typing, Ping, Pong, Error,
	}
	for _, typ := range types {
		t.Run(mustName(typ), func(t *testing.T) {
			f := Frame{Type: typ, ID: "x", Payload: json.RawMessage(`{}`)}
			data, err := json.Marshal(f)
			if err != nil {
				t.Fatalf("Marshal: %v", err)
			}
			var f2 Frame
			if err := json.Unmarshal(data, &f2); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if f2.Type != typ {
				t.Errorf("Type = %d, want %d", f2.Type, typ)
			}
		})
	}
}

func TestFrame_NullPayload(t *testing.T) {
	// JSON null payload is stored by encoding/json as the literal bytes "null".
	data := `{"type":1,"id":"xyz","payload":null}`
	var f Frame
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if f.Type != MsgSend {
		t.Errorf("Type = %d, want %d", f.Type, MsgSend)
	}
	if f.ID != "xyz" {
		t.Errorf("ID = %q, want %q", f.ID, "xyz")
	}
	if string(f.Payload) != "null" {
		t.Errorf("Payload = %q, want %q", string(f.Payload), "null")
	}
}

func TestFrame_EmptyPayload(t *testing.T) {
	data := `{"type":61,"id":"ping-1","payload":{}}`
	var f Frame
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if f.Type != Ping {
		t.Errorf("Type = %d, want %d", f.Type, Ping)
	}
}

func TestFrame_MissingOptionalFields(t *testing.T) {
	// Only "type" is required for unmarshal to succeed without error.
	data := `{"type":71}`
	var f Frame
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if f.Type != Error {
		t.Errorf("Type = %d, want %d", f.Type, Error)
	}
	if f.ID != "" {
		t.Errorf("ID = %q, want empty", f.ID)
	}
	if f.Payload != nil {
		t.Errorf("Payload = %v, want nil", f.Payload)
	}
}

// ---------------------------------------------------------------------------
// Payload structs – marshal / unmarshal
// ---------------------------------------------------------------------------

func TestPayload_MsgSendPayload(t *testing.T) {
	t.Run("marshal_unmarshal", func(t *testing.T) {
		in := MsgSendPayload{
			ConvID:      "conv-1",
			ContentType: 1,
			Body:        "hello",
			ReplyTo:     0,
			ClientSeq:   100,
			Mention:     []string{"user-a", "user-b"},
		}
		runPayloadTest(t, in)
	})

	t.Run("omit_empty_mention", func(t *testing.T) {
		in := MsgSendPayload{
			ConvID:      "conv-1",
			ContentType: 1,
			Body:        "hello",
			ReplyTo:     0,
			ClientSeq:   100,
		}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if hasKey(t, data, "mention") {
			t.Error("mention should be omitted when empty")
		}
	})
}

func TestPayload_MsgSendAckPayload(t *testing.T) {
	in := MsgSendAckPayload{
		MsgID:     42,
		Timestamp: 1000000,
		ClientSeq: 100,
		Status:    0,
	}
	runPayloadTest(t, in)
}

func TestPayload_MsgPushPayload(t *testing.T) {
	t.Run("marshal_unmarshal", func(t *testing.T) {
		in := MsgPushPayload{
			MsgID:       99,
			ConvID:      "conv-2",
			SenderID:    "user-1",
			ContentType: 2,
			Body:        "push body",
			ReplyTo:     5,
			Mention:     []string{"user-x"},
			Timestamp:   2000000,
			ConvSeq:     10,
		}
		runPayloadTest(t, in)
	})

	t.Run("omit_empty_mention", func(t *testing.T) {
		in := MsgPushPayload{
			MsgID:       1,
			ConvID:      "c",
			SenderID:    "u",
			ContentType: 0,
			Body:        "b",
			Timestamp:   1,
			ConvSeq:     1,
		}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if hasKey(t, data, "mention") {
			t.Error("mention should be omitted when empty")
		}
	})
}

func TestPayload_MsgReceivedPayload(t *testing.T) {
	in := MsgReceivedPayload{
		MsgID:   77,
		ConvID:  "conv-3",
		ConvSeq: 12,
	}
	runPayloadTest(t, in)
}

func TestPayload_SyncReqPayload(t *testing.T) {
	in := SyncReqPayload{
		ConvID:      "conv-4",
		LastConvSeq: 50,
		Limit:       20,
	}
	runPayloadTest(t, in)
}

func TestPayload_SyncMessage(t *testing.T) {
	in := SyncMessage{
		MsgID:       1,
		SenderID:    "user-a",
		ContentType: 1,
		Body:        "sync msg",
		Timestamp:   3000,
		ConvSeq:     15,
	}
	runPayloadTest(t, in)
}

func TestPayload_SyncResPayload(t *testing.T) {
	in := SyncResPayload{
		ConvID: "conv-5",
		Messages: []SyncMessage{
			{MsgID: 1, SenderID: "u1", ContentType: 1, Body: "m1", Timestamp: 100, ConvSeq: 1},
			{MsgID: 2, SenderID: "u2", ContentType: 2, Body: "m2", Timestamp: 200, ConvSeq: 2},
		},
		HasMore: true,
	}
	runPayloadTest(t, in)
}

func TestPayload_SyncResPayload_EmptyMessages(t *testing.T) {
	in := SyncResPayload{
		ConvID:   "conv-5",
		Messages: []SyncMessage{},
		HasMore:  false,
	}
	runPayloadTest(t, in)
}

func TestPayload_MsgReadNotifyPayload(t *testing.T) {
	in := MsgReadNotifyPayload{
		ConvID:    "conv-6",
		UserID:    "user-r",
		SessionID: "sess-1",
		MsgID:     200,
		Timestamp: 4000,
	}
	runPayloadTest(t, in)
}

func TestPayload_SessionEventPayload(t *testing.T) {
	t.Run("with_device_fields", func(t *testing.T) {
		in := SessionEventPayload{
			UserID:     "user-s",
			SessionID:  "sess-2",
			Device:     1,
			DeviceName: "ios",
		}
		runPayloadTest(t, in)
	})

	t.Run("omit_device_fields", func(t *testing.T) {
		in := SessionEventPayload{
			UserID:    "user-s",
			SessionID: "sess-2",
		}
		data, err := json.Marshal(in)
		if err != nil {
			t.Fatalf("Marshal: %v", err)
		}
		if hasKey(t, data, "device") {
			t.Error("device should be omitted when zero")
		}
		if hasKey(t, data, "device_name") {
			t.Error("device_name should be omitted when empty")
		}
	})
}

func TestPayload_SessionRecoverPayload(t *testing.T) {
	in := SessionRecoverPayload{
		SessionID: "sess-recover",
	}
	runPayloadTest(t, in)
}

func TestPayload_SessionRecoverAckPayload(t *testing.T) {
	in := SessionRecoverAckPayload{
		SessionID: "sess-ack",
		UserID:    "user-ack",
		Timestamp: 5000,
	}
	runPayloadTest(t, in)
}

func TestPayload_TypingPayload(t *testing.T) {
	in := TypingPayload{
		ConvID:    "conv-typing",
		UserID:    "user-type",
		SessionID: "sess-type",
	}
	runPayloadTest(t, in)
}

func TestPayload_EmptyPayload(t *testing.T) {
	in := EmptyPayload{}
	runPayloadTest(t, in)
}

func TestPayload_ErrorPayload(t *testing.T) {
	in := ErrorPayload{
		Code:    401,
		Message: "unauthorized",
	}
	runPayloadTest(t, in)
}

func TestPayload_ErrorPayload_Zero(t *testing.T) {
	in := ErrorPayload{}
	runPayloadTest(t, in)
}

// ---------------------------------------------------------------------------
// Frame with concrete payload types – composite round-trip
// ---------------------------------------------------------------------------

func TestFrame_WithMsgSendPayload(t *testing.T) {
	payload := MsgSendPayload{
		ConvID:      "conv-frame",
		ContentType: 3,
		Body:        "frame body",
		ClientSeq:   200,
	}
	frame := mustFrame(t, MsgSend, "f1", payload)

	var got MsgSendPayload
	mustDecodePayload(t, frame.Payload, &got)

	assertMsgSendPayloadEqual(t, got, payload)
}

func TestFrame_WithMsgSendPayload_WithMention(t *testing.T) {
	payload := MsgSendPayload{
		ConvID:      "conv-frame",
		ContentType: 3,
		Body:        "frame body",
		ClientSeq:   200,
		Mention:     []string{"user-a", "user-b"},
	}
	frame := mustFrame(t, MsgSend, "f1", payload)

	var got MsgSendPayload
	mustDecodePayload(t, frame.Payload, &got)

	assertMsgSendPayloadEqual(t, got, payload)
}

func TestFrame_WithMsgSendAckPayload(t *testing.T) {
	payload := MsgSendAckPayload{MsgID: 1, Timestamp: 100, ClientSeq: 10, Status: 1}
	frame := mustFrame(t, MsgSendAck, "ack-1", payload)

	var got MsgSendAckPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("MsgSendAckPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithMsgPushPayload(t *testing.T) {
	payload := MsgPushPayload{
		MsgID: 5, ConvID: "c", SenderID: "u",
		ContentType: 1, Body: "push", Timestamp: 99, ConvSeq: 3,
	}
	frame := mustFrame(t, MsgPush, "push-1", payload)

	var got MsgPushPayload
	mustDecodePayload(t, frame.Payload, &got)

	assertMsgPushPayloadEqual(t, got, payload)
}

func TestFrame_WithMsgPushPayload_WithMention(t *testing.T) {
	payload := MsgPushPayload{
		MsgID: 5, ConvID: "c", SenderID: "u",
		ContentType: 1, Body: "push", Mention: []string{"@all"},
		Timestamp: 99, ConvSeq: 3,
	}
	frame := mustFrame(t, MsgPush, "push-1", payload)

	var got MsgPushPayload
	mustDecodePayload(t, frame.Payload, &got)

	assertMsgPushPayloadEqual(t, got, payload)
}

func TestFrame_WithMsgReceivedPayload(t *testing.T) {
	payload := MsgReceivedPayload{MsgID: 10, ConvID: "c", ConvSeq: 5}
	frame := mustFrame(t, MsgReceived, "recv-1", payload)

	var got MsgReceivedPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("MsgReceivedPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithSyncReqPayload(t *testing.T) {
	payload := SyncReqPayload{ConvID: "c", LastConvSeq: 100, Limit: 50}
	frame := mustFrame(t, SyncReq, "sync-req-1", payload)

	var got SyncReqPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("SyncReqPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithSyncResPayload(t *testing.T) {
	payload := SyncResPayload{
		ConvID:   "c",
		HasMore:  true,
		Messages: []SyncMessage{{MsgID: 1, SenderID: "u", Body: "m", Timestamp: 1, ConvSeq: 1}},
	}
	frame := mustFrame(t, SyncRes, "sync-res-1", payload)

	var got SyncResPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got.ConvID != payload.ConvID || got.HasMore != payload.HasMore {
		t.Errorf("SyncResPayload mismatch: got %+v, want %+v", got, payload)
	}
	if len(got.Messages) != len(payload.Messages) {
		t.Errorf("Messages length = %d, want %d", len(got.Messages), len(payload.Messages))
	} else if got.Messages[0] != payload.Messages[0] {
		t.Errorf("Messages[0] mismatch: got %+v, want %+v", got.Messages[0], payload.Messages[0])
	}
}

func TestFrame_WithMsgReadNotifyPayload(t *testing.T) {
	payload := MsgReadNotifyPayload{ConvID: "c", UserID: "u", SessionID: "s", MsgID: 7, Timestamp: 88}
	frame := mustFrame(t, MsgReadNotify, "read-1", payload)

	var got MsgReadNotifyPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("MsgReadNotifyPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithSessionEventPayload(t *testing.T) {
	payload := SessionEventPayload{UserID: "u", SessionID: "s", Device: 2, DeviceName: "android"}
	frame := mustFrame(t, SessionOnline, "online-1", payload)

	var got SessionEventPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("SessionEventPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithSessionRecoverPayload(t *testing.T) {
	payload := SessionRecoverPayload{SessionID: "s-recover"}
	frame := mustFrame(t, SessionRecover, "recover-1", payload)

	var got SessionRecoverPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("SessionRecoverPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithSessionRecoverAckPayload(t *testing.T) {
	payload := SessionRecoverAckPayload{SessionID: "s-ack", UserID: "u-ack", Timestamp: 123}
	frame := mustFrame(t, SessionRecoverAck, "recover-ack-1", payload)

	var got SessionRecoverAckPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("SessionRecoverAckPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithTypingPayload(t *testing.T) {
	payload := TypingPayload{ConvID: "c", UserID: "u", SessionID: "s"}
	frame := mustFrame(t, Typing, "typing-1", payload)

	var got TypingPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("TypingPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithErrorPayload(t *testing.T) {
	payload := ErrorPayload{Code: 403, Message: "forbidden"}
	frame := mustFrame(t, Error, "err-1", payload)

	var got ErrorPayload
	mustDecodePayload(t, frame.Payload, &got)

	if got != payload {
		t.Errorf("ErrorPayload mismatch: got %+v, want %+v", got, payload)
	}
}

func TestFrame_WithEmptyPayload(t *testing.T) {
	frame := mustFrame(t, Ping, "ping-1", EmptyPayload{})

	// An EmptyPayload serialises to "{}". Verify that it round-trips.
	if string(frame.Payload) != `{}` {
		t.Errorf("EmptyPayload JSON = %s, want {}", string(frame.Payload))
	}

	var got EmptyPayload
	mustDecodePayload(t, frame.Payload, &got)
}

func TestFrame_ZeroValue(t *testing.T) {
	var f Frame
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal zero Frame: %v", err)
	}
	var f2 Frame
	if err := json.Unmarshal(data, &f2); err != nil {
		t.Fatalf("Unmarshal zero Frame: %v", err)
	}
	if f2.Type != 0 {
		t.Errorf("Type = %d, want 0", f2.Type)
	}
	if f2.ID != "" {
		t.Errorf("ID = %q, want empty", f2.ID)
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestFrame_UnknownMessageTypeInJSON(t *testing.T) {
	data := `{"type":999,"id":"unknown","payload":{"code":1}}`
	var f Frame
	if err := json.Unmarshal([]byte(data), &f); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if f.Type != 999 {
		t.Errorf("Type = %d, want 999", f.Type)
	}
}

func TestPayload_DeepEqualSyncMessage(t *testing.T) {
	in := SyncMessage{
		MsgID: 1, SenderID: "u",
		ContentType: 2, Body: "hello",
		Timestamp: 12345, ConvSeq: 6,
	}
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var out SyncMessage
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if out != in {
		t.Errorf("SyncMessage mismatch: got %+v, want %+v", out, in)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// runPayloadTest marshals in, unmarshals into a new value of the same type,
// and compares the result to in.
func runPayloadTest[T any](t *testing.T, in T) {
	t.Helper()

	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("Marshal %T: %v", in, err)
	}

	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("Unmarshal %T: %v (input=%s)", in, err, string(data))
	}

	// Re-marshal the decoded value to compare structurally.
	data2, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("Re-marshal %T: %v", out, err)
	}

	if string(data) != string(data2) {
		t.Errorf("%T round-trip mismatch:\n  got:  %s\n  want: %s", in, string(data2), string(data))
	}
}

// mustFrame creates a Frame, marshals it, unmarshals it, and returns the
// decoded Frame. It fails the test on error.
func mustFrame(t *testing.T, typ MessageType, id string, payload any) Frame {
	t.Helper()

	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal payload: %v", err)
	}

	f := Frame{Type: typ, ID: id, Payload: raw}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal Frame: %v", err)
	}

	var f2 Frame
	if err := json.Unmarshal(data, &f2); err != nil {
		t.Fatalf("Unmarshal Frame: %v", err)
	}

	if f2.Type != typ {
		t.Errorf("Frame.Type = %d, want %d", f2.Type, typ)
	}
	if f2.ID != id {
		t.Errorf("Frame.ID = %q, want %q", f2.ID, id)
	}
	return f2
}

// mustDecodePayload unmarshals raw into v and fails the test on error.
func mustDecodePayload[T any](t *testing.T, raw json.RawMessage, v *T) {
	t.Helper()
	if err := json.Unmarshal(raw, v); err != nil {
		t.Fatalf("Decode payload %T: %v (raw=%s)", *v, err, string(raw))
	}
}

// mustName returns the constant name for a known MessageType, or "unknown".
func mustName(typ MessageType) string {
	switch typ {
	case MsgSend:
		return "MsgSend"
	case MsgSendAck:
		return "MsgSendAck"
	case MsgPush:
		return "MsgPush"
	case MsgReceived:
		return "MsgReceived"
	case SyncReq:
		return "SyncReq"
	case SyncRes:
		return "SyncRes"
	case MsgReadNotify:
		return "MsgReadNotify"
	case SessionOnline:
		return "SessionOnline"
	case SessionOffline:
		return "SessionOffline"
	case SessionRecover:
		return "SessionRecover"
	case SessionRecoverAck:
		return "SessionRecoverAck"
	case Typing:
		return "Typing"
	case Ping:
		return "Ping"
	case Pong:
		return "Pong"
	case Error:
		return "Error"
	default:
		return "unknown"
	}
}

// assertMsgSendPayloadEqual compares two MsgSendPayload values field by field,
// including the Mention slice.
func assertMsgSendPayloadEqual(t *testing.T, got, want MsgSendPayload) {
	t.Helper()
	if got.ConvID != want.ConvID {
		t.Errorf("ConvID = %q, want %q", got.ConvID, want.ConvID)
	}
	if got.ContentType != want.ContentType {
		t.Errorf("ContentType = %d, want %d", got.ContentType, want.ContentType)
	}
	if got.Body != want.Body {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
	if got.ReplyTo != want.ReplyTo {
		t.Errorf("ReplyTo = %d, want %d", got.ReplyTo, want.ReplyTo)
	}
	if got.ClientSeq != want.ClientSeq {
		t.Errorf("ClientSeq = %d, want %d", got.ClientSeq, want.ClientSeq)
	}
	if len(got.Mention) != len(want.Mention) {
		t.Errorf("Mention length = %d, want %d; got=%v want=%v", len(got.Mention), len(want.Mention), got.Mention, want.Mention)
	} else {
		for i := range got.Mention {
			if got.Mention[i] != want.Mention[i] {
				t.Errorf("Mention[%d] = %q, want %q", i, got.Mention[i], want.Mention[i])
			}
		}
	}
}

// assertMsgPushPayloadEqual compares two MsgPushPayload values field by field,
// including the Mention slice.
func assertMsgPushPayloadEqual(t *testing.T, got, want MsgPushPayload) {
	t.Helper()
	if got.MsgID != want.MsgID {
		t.Errorf("MsgID = %d, want %d", got.MsgID, want.MsgID)
	}
	if got.ConvID != want.ConvID {
		t.Errorf("ConvID = %q, want %q", got.ConvID, want.ConvID)
	}
	if got.SenderID != want.SenderID {
		t.Errorf("SenderID = %q, want %q", got.SenderID, want.SenderID)
	}
	if got.ContentType != want.ContentType {
		t.Errorf("ContentType = %d, want %d", got.ContentType, want.ContentType)
	}
	if got.Body != want.Body {
		t.Errorf("Body = %q, want %q", got.Body, want.Body)
	}
	if got.ReplyTo != want.ReplyTo {
		t.Errorf("ReplyTo = %d, want %d", got.ReplyTo, want.ReplyTo)
	}
	if got.Timestamp != want.Timestamp {
		t.Errorf("Timestamp = %d, want %d", got.Timestamp, want.Timestamp)
	}
	if got.ConvSeq != want.ConvSeq {
		t.Errorf("ConvSeq = %d, want %d", got.ConvSeq, want.ConvSeq)
	}
	if len(got.Mention) != len(want.Mention) {
		t.Errorf("Mention length = %d, want %d; got=%v want=%v", len(got.Mention), len(want.Mention), got.Mention, want.Mention)
	} else {
		for i := range got.Mention {
			if got.Mention[i] != want.Mention[i] {
				t.Errorf("Mention[%d] = %q, want %q", i, got.Mention[i], want.Mention[i])
			}
		}
	}
}

// hasKey reports whether raw JSON data contains the given key as a top-level
// key. It is only used for omitempty assertions and is not a full JSON parser.
func hasKey(t *testing.T, data []byte, key string) bool {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("unmarshal to map: %v", err)
	}
	_, ok := m[key]
	return ok
}
