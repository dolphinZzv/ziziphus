package tasks

import (
	"encoding/json"
	"testing"

	"ziziphus/pkg/model"
)

func TestNewWebhookForwardTask(t *testing.T) {
	wh := &model.ConvWebhook{
		ID:          42,
		ConvID:      "conv_1",
		Name:        "test-webhook",
		APIKeyPlain: "test-api-key",
		APIKeyHash:  "test-hash",
		CallbackURL: "https://example.com/hook",
		Headers: []model.WebhookHeader{
			{Key: "X-Custom", Value: "val"},
		},
	}

	msg := &model.Message{
		MsgID:       100,
		ConvID:      "conv_1",
		SenderID:    "user_1",
		SenderName:  "Alice",
		ContentType: model.ContentText,
		Body:        "hello",
		ReplyTo:     0,
		Timestamp:   1000,
	}

	payload, err := NewWebhookForwardTask(wh, "TestApp", msg)
	if err != nil {
		t.Fatalf("NewWebhookForwardTask: %v", err)
	}

	if payload.CallbackURL != "https://example.com/hook" {
		t.Errorf("CallbackURL = %q, want %q", payload.CallbackURL, "https://example.com/hook")
	}
	if payload.APIKeyHash != "test-hash" {
		t.Errorf("APIKeyHash = %q, want %q", payload.APIKeyHash, "test-hash")
	}
	if payload.WhID != 42 {
		t.Errorf("WhID = %d, want 42", payload.WhID)
	}
	if payload.MsgID != 100 {
		t.Errorf("MsgID = %d, want 100", payload.MsgID)
	}
	if payload.SenderName != "Alice" {
		t.Errorf("SenderName = %q, want %q", payload.SenderName, "Alice")
	}
	if len(payload.Headers) != 1 || payload.Headers[0].Key != "X-Custom" {
		t.Errorf("Headers not preserved correctly")
	}

	// Test marshaling
	data, err := payload.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("Marshal returned empty data")
	}

	// Verify it round-trips
	var decoded WebhookForwardPayload
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if decoded.SenderName != "Alice" {
		t.Errorf("round-trip SenderName = %q, want %q", decoded.SenderName, "Alice")
	}
}
