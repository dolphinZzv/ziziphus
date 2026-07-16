package tasks

import (
	"encoding/json"
	"testing"
)

func TestNewEmailVerificationTask(t *testing.T) {
	b, err := NewEmailVerificationTask("user@example.com", "123456")
	if err != nil {
		t.Fatalf("NewEmailVerificationTask: %v", err)
	}
	var payload EmailVerificationPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.To != "user@example.com" {
		t.Errorf("To = %q, want %q", payload.To, "user@example.com")
	}
	if payload.Code != "123456" {
		t.Errorf("Code = %q, want %q", payload.Code, "123456")
	}
}

func TestNewPasswordResetTask(t *testing.T) {
	b, err := NewPasswordResetTask("user@example.com", "654321")
	if err != nil {
		t.Fatalf("NewPasswordResetTask: %v", err)
	}
	var payload PasswordResetPayload
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if payload.To != "user@example.com" {
		t.Errorf("To = %q, want %q", payload.To, "user@example.com")
	}
	if payload.Code != "654321" {
		t.Errorf("Code = %q, want %q", payload.Code, "654321")
	}
}

func TestMailDispatcher_Disabled(t *testing.T) {
	d := NewMailDispatcher(nil, false)
	if d.Enabled() {
		t.Error("expected disabled dispatcher")
	}
	if err := d.SendVerificationCode("a@b.com", "123"); err == nil {
		t.Error("expected error from disabled dispatcher")
	}
	if err := d.SendPasswordResetCode("a@b.com", "123"); err == nil {
		t.Error("expected error from disabled dispatcher")
	}
}
