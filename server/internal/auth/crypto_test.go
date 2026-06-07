package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

const maxOAEPPlaintext = 190 // 2048-bit RSA OAEP SHA-256: 256 - 2*32 - 2

func TestGenerateKeyPair(t *testing.T) {
	priv, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair() returned error: %v", err)
	}
	if priv == nil {
		t.Fatal("GenerateKeyPair() returned nil key")
	}
	if priv.N.BitLen() != 2048 {
		t.Errorf("expected key size 2048 bits, got %d", priv.N.BitLen())
	}
	// Verify it is a valid key by using it for encryption
	pub := &priv.PublicKey
	msg := []byte("hello")
	ciphertext, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, pub, msg, nil)
	if err != nil {
		t.Fatalf("encrypt with generated key failed: %v", err)
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, priv, ciphertext, nil)
	if err != nil {
		t.Fatalf("decrypt with generated key failed: %v", err)
	}
	if string(plaintext) != "hello" {
		t.Errorf("round-trip produced %q, want %q", plaintext, "hello")
	}
}

func TestNewCryptoFromKeys(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	if c == nil {
		t.Fatal("NewCryptoFromKeys returned nil")
	}
	// Verify it can encrypt and decrypt
	ctx := context.Background()
	msg := []byte("test data")
	ciphertext, err := c.Encrypt(ctx, msg)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	plaintext, err := c.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if string(plaintext) != "test data" {
		t.Errorf("round-trip produced %q, want %q", plaintext, "test data")
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	ctx := context.Background()

	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"single byte", []byte{0x42}},
		{"short string", []byte("hello world")},
		{"lorem ipsum", []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit.")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ciphertext, err := c.Encrypt(ctx, tt.data)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}
			plaintext, err := c.Decrypt(ctx, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}
			if string(plaintext) != string(tt.data) {
				t.Errorf("Decrypt returned %q, want %q", plaintext, tt.data)
			}
		})
	}
}

func TestEncrypt_DifferentSizes(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	ctx := context.Background()

	tests := []struct {
		name string
		size int
	}{
		{"small (1 byte)", 1},
		{"medium (100 bytes)", 100},
		{"large (190 bytes)", maxOAEPPlaintext},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext := make([]byte, tt.size)
			_, err := rand.Read(plaintext)
			if err != nil {
				t.Fatalf("rand.Read failed: %v", err)
			}

			ciphertext, err := c.Encrypt(ctx, plaintext)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}
			result, err := c.Decrypt(ctx, ciphertext)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}
			if string(result) != string(plaintext) {
				t.Errorf("round-trip mismatch for %d bytes", tt.size)
			}
		})
	}
}

func TestEncrypt_ExceedsMaxSize(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	ctx := context.Background()

	// 191 bytes exceeds max plaintext for RSA 2048 OAEP SHA-256
	large := make([]byte, maxOAEPPlaintext+1)
	_, err = rand.Read(large)
	if err != nil {
		t.Fatalf("rand.Read failed: %v", err)
	}

	_, err = c.Encrypt(ctx, large)
	if err == nil {
		t.Error("expected error when encrypting data that exceeds max plaintext size, got nil")
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	priv1, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	priv2, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	c1 := NewCryptoFromKeys(priv1)
	c2 := NewCryptoFromKeys(priv2)
	ctx := context.Background()

	plaintext := []byte("secret message")
	ciphertext, err := c1.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	_, err = c2.Decrypt(ctx, ciphertext)
	if err == nil {
		t.Error("expected error when decrypting with wrong key, got nil")
	}
}

func TestEncrypt_NonDeterministic(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	ctx := context.Background()

	plaintext := []byte("same data")
	ciphertext1, err := c.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("first Encrypt failed: %v", err)
	}
	ciphertext2, err := c.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("second Encrypt failed: %v", err)
	}

	if len(ciphertext1) != len(ciphertext2) {
		t.Fatal("ciphertexts have different lengths, cannot compare")
	}
	equal := true
	for i := range ciphertext1 {
		if ciphertext1[i] != ciphertext2[i] {
			equal = false
			break
		}
	}
	if equal {
		t.Error("encrypting same data produced identical ciphertext; expected non-deterministic output")
	}
}

func TestGenerateAndSaveKeys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "private.pem")

	err := GenerateAndSaveKeys(path)
	if err != nil {
		t.Fatalf("GenerateAndSaveKeys failed: %v", err)
	}

	// Verify file was created and is not empty
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("os.Stat failed: %v", err)
	}
	if info.Size() == 0 {
		t.Fatal("generated key file is empty")
	}

	// Verify file contains valid PEM
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("key file is empty")
	}

	// Verify the file can be loaded by NewCrypto
	c, err := NewCrypto(path)
	if err != nil {
		t.Fatalf("NewCrypto from generated file failed: %v", err)
	}
	if c == nil {
		t.Fatal("NewCrypto returned nil")
	}

	// Verify the loaded key works for encrypt/decrypt
	ctx := context.Background()
	msg := []byte("round-trip from generated key file")
	ciphertext, err := c.Encrypt(ctx, msg)
	if err != nil {
		t.Fatalf("Encrypt with loaded key failed: %v", err)
	}
	plaintext, err := c.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt with loaded key failed: %v", err)
	}
	if string(plaintext) != string(msg) {
		t.Errorf("round-trip produced %q, want %q", plaintext, msg)
	}
}

func TestNewCrypto_FromGeneratedFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "key.pem")

	err := GenerateAndSaveKeys(path)
	if err != nil {
		t.Fatalf("GenerateAndSaveKeys failed: %v", err)
	}

	c, err := NewCrypto(path)
	if err != nil {
		t.Fatalf("NewCrypto failed: %v", err)
	}
	if c == nil {
		t.Fatal("NewCrypto returned nil")
	}

	// Encrypt and decrypt with the loaded key
	ctx := context.Background()
	original := []byte("data from file-backed crypto")
	ciphertext, err := c.Encrypt(ctx, original)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	result, err := c.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}
	if string(result) != string(original) {
		t.Errorf("got %q, want %q", result, original)
	}
}
