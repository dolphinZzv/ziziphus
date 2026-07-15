package auth

import (
	"encoding/base32"
	"strings"
	"testing"
	"time"
)

func TestGenerateTOTPSecret_Length(t *testing.T) {
	secret := GenerateTOTPSecret()
	if len(secret) < 16 {
		t.Errorf("secret too short: %d", len(secret))
	}
	// Should be base32 decodable
	_, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		t.Errorf("secret not base32: %v", err)
	}
}

func TestGenerateTOTPSecret_Unique(t *testing.T) {
	s1 := GenerateTOTPSecret()
	s2 := GenerateTOTPSecret()
	if s1 == s2 {
		t.Error("secrets should be unique")
	}
}

func TestTOTPURI_Format(t *testing.T) {
	uri := TOTPURI("testuser", "Ziziphus", "SECRETKEY123")
	if !strings.HasPrefix(uri, "otpauth://totp/") {
		t.Errorf("bad prefix: %s", uri)
	}
	if !strings.Contains(uri, "secret=SECRETKEY123") {
		t.Errorf("missing secret: %s", uri)
	}
	if !strings.Contains(uri, "algorithm=SHA1") {
		t.Errorf("missing algorithm: %s", uri)
	}
}

func TestVerifyTOTP_ValidCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)

	now := time.Now().Unix()
	step := now / 30
	code := totpCode(key, step)

	valid := VerifyTOTP(secret, code)
	if !valid {
		t.Error("valid code rejected")
	}
}

func TestVerifyTOTP_ClockSkew(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	key, _ := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)

	now := time.Now().Unix()
	step := now / 30
	currentCode := totpCode(key, step)
	prevCode := totpCode(key, step-1)
	nextCode := totpCode(key, step+1)

	if !VerifyTOTP(secret, currentCode) {
		t.Error("current code rejected")
	}
	if !VerifyTOTP(secret, prevCode) {
		t.Error("previous code rejected (clock skew)")
	}
	if !VerifyTOTP(secret, nextCode) {
		t.Error("next code rejected (clock skew)")
	}
}

func TestVerifyTOTP_WrongCode(t *testing.T) {
	secret := "JBSWY3DPEHPK3PXP"
	wrongCode := "000000"
	if VerifyTOTP(secret, wrongCode) {
		t.Error("wrong code accepted")
	}
	if VerifyTOTP(secret, "") {
		t.Error("empty code accepted")
	}
	if VerifyTOTP(secret, "abc123") {
		t.Error("non-numeric code accepted")
	}
	if VerifyTOTP(secret, "12345") {
		t.Error("5-digit code accepted")
	}
}

func TestVerifyTOTP_InvalidSecret(t *testing.T) {
	if VerifyTOTP("!!!invalid!!!", "123456") {
		t.Error("invalid secret should return false")
	}
}

func TestGenerateEmailOTP_Format(t *testing.T) {
	code := GenerateEmailOTP()
	if len(code) != 6 {
		t.Errorf("expected 6 digits, got %d: %s", len(code), code)
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			t.Errorf("non-digit char in OTP: %c", c)
			break
		}
	}
}

func TestGenerateEmailOTP_Unique(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 10; i++ {
		code := GenerateEmailOTP()
		if codes[code] {
			// Could be collision but unlikely in 10 tries
			continue
		}
		codes[code] = true
	}
	if len(codes) < 3 {
		t.Error("too many collisions in OTPs")
	}
}

