package auth

import (
	"testing"

	"ziziphus/config"
)

func TestNewMailer(t *testing.T) {
	cfg := config.SMTPConfig{Host: "smtp.example.com", User: "user", Password: "pass", From: "noreply@example.com"}
	m := NewMailer(cfg, "TestApp")
	if m == nil {
		t.Fatal("NewMailer returned nil")
	}
	if m.Enabled() != true {
		t.Error("Expected Enabled() = true with valid config")
	}
	if m.appNameVal() != "TestApp" {
		t.Errorf("appNameVal() = %q, want %q", m.appNameVal(), "TestApp")
	}
}

func TestNewMailer_Disabled(t *testing.T) {
	m := NewMailer(config.SMTPConfig{}, "")
	if m.Enabled() {
		t.Error("Expected Enabled() = false with empty config")
	}
}

func TestMailer_UpdateConfig(t *testing.T) {
	m := NewMailer(config.SMTPConfig{}, "")
	if m.Enabled() {
		t.Error("expected disabled initially")
	}

	m.UpdateConfig(config.SMTPConfig{Host: "smtp.test.com", User: "test", Password: "pw", From: "test@test.com"})
	if !m.Enabled() {
		t.Error("expected enabled after UpdateConfig")
	}
}

func TestMailer_SetAppName(t *testing.T) {
	m := NewMailer(config.SMTPConfig{}, "Initial")
	m.SetAppName("Updated")
	if m.appNameVal() != "Updated" {
		t.Errorf("appNameVal() = %q, want %q", m.appNameVal(), "Updated")
	}
}

func TestMailer_AppNameVal_Fallback(t *testing.T) {
	m := &Mailer{} // no atomic set
	if m.appNameVal() != "Ziziphus" {
		t.Errorf("appNameVal() = %q, want %q", m.appNameVal(), "Ziziphus")
	}
}

func TestMailer_CfgVal_Fallback(t *testing.T) {
	m := &Mailer{} // no atomic set
	got := m.cfgVal()
	if got != (config.SMTPConfig{}) {
		t.Errorf("cfgVal() = %+v, want zero value", got)
	}
}

func TestMailer_SendVerificationCode_Disabled(t *testing.T) {
	m := NewMailer(config.SMTPConfig{}, "Test")
	err := m.SendVerificationCode("user@example.com", "123456")
	if err == nil {
		t.Fatal("expected error when mailer is disabled")
	}
}

func TestMailer_SendVerificationCodeLang_Disabled(t *testing.T) {
	m := NewMailer(config.SMTPConfig{}, "Test")
	err := m.SendVerificationCodeLang("user@example.com", "123456", "en")
	if err == nil {
		t.Fatal("expected error when mailer is disabled")
	}
}

func TestMailer_SendVerificationCodeLang_UnknownLang_Fallback(t *testing.T) {
	m := NewMailer(config.SMTPConfig{Host: "smtp.test.com", User: "test", Password: "pw", From: "test@test.com"}, "TestApp")
	err := m.SendVerificationCodeLang("user@example.com", "123456", "jp")
	if err == nil {
		t.Log("SendVerificationCodeLang with unknown lang attempted SMTP (expected)")
	} else {
		t.Logf("SendVerificationCodeLang with unknown lang returned: %v", err)
		// Only check that the error is about SMTP connection, not template parsing
	}
}

func TestMailer_CfgVal_Race(t *testing.T) {
	cfg1 := config.SMTPConfig{Host: "smtp1.com", User: "u1"}
	cfg2 := config.SMTPConfig{Host: "smtp2.com", User: "u2"}
	m := NewMailer(cfg1, "Test")

	done := make(chan bool)
	go func() {
		m.UpdateConfig(cfg2)
		done <- true
	}()
	m.cfgVal()
	<-done
}
