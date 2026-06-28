package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
	"time"
)

// GenerateTOTPSecret generates a random base32-encoded secret for TOTP.
func GenerateTOTPSecret() string {
	b := make([]byte, 20)
	rand.Read(b)
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b)
}

// TOTPURI generates the otpauth:// URI for QR codes.
func TOTPURI(account, issuer, secret string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
		issuer, account, secret, issuer)
}

// VerifyTOTP checks a TOTP code against the secret.
func VerifyTOTP(secret, code string) bool {
	if len(code) != 6 {
		return false
	}
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(secret)
	if err != nil {
		return false
	}
	now := time.Now().Unix()
	// Allow ±1 step (30s each) for clock skew
	for step := now/30 - 1; step <= now/30+1; step++ {
		if totpCode(key, step) == code {
			return true
		}
	}
	return false
}

func totpCode(key []byte, counter int64) string {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	hash := mac.Sum(nil)
	offset := hash[len(hash)-1] & 0x0F
	binary := binary.BigEndian.Uint32(hash[offset:offset+4]) & 0x7FFFFFFF
	otp := int(binary) % int(math.Pow10(6))
	return fmt.Sprintf("%06d", otp)
}

// GenerateEmailOTP generates a 6-digit OTP for email verification.
func GenerateEmailOTP() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%06d", binary.BigEndian.Uint32(b)%1000000)
}
