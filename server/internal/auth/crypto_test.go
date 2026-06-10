package auth

import (
	"testing"
)

func TestHashPassword_Success(t *testing.T) {
	hash, err := HashPassword("my-secret-password")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword returned empty string")
	}
	if hash == "my-secret-password" {
		t.Fatal("HashPassword returned plaintext password")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	password := "correct-horse-battery-staple"
	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if !CheckPassword(password, hash) {
		t.Error("CheckPassword returned false for correct password")
	}
}

func TestCheckPassword_Wrong(t *testing.T) {
	hash, err := HashPassword("real-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if CheckPassword("wrong-password", hash) {
		t.Error("CheckPassword returned true for wrong password")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, err := HashPassword("real-password")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}

	if CheckPassword("", hash) {
		t.Error("CheckPassword returned true for empty password")
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	if CheckPassword("password", "not-a-valid-hash") {
		t.Error("CheckPassword returned true for invalid hash")
	}
}

func TestHashPassword_EmptyInput(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword('') returned error: %v", err)
	}
	if hash == "" {
		t.Fatal("HashPassword('') returned empty string")
	}
}

func TestHashPassword_DifferentHashes(t *testing.T) {
	hash1, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("first HashPassword: %v", err)
	}
	hash2, err := HashPassword("same-password")
	if err != nil {
		t.Fatalf("second HashPassword: %v", err)
	}
	// bcrypt uses a random salt, so two hashes of the same password must differ
	if hash1 == hash2 {
		t.Error("two hashes of the same password are identical; expected different salts")
	}
	// Both hashes should still verify correctly
	if !CheckPassword("same-password", hash1) {
		t.Error("CheckPassword failed on first hash")
	}
	if !CheckPassword("same-password", hash2) {
		t.Error("CheckPassword failed on second hash")
	}
}

func TestHashPassword_MinCost(t *testing.T) {
	hash, err := HashPassword("long-password-12345")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword("long-password-12345", hash) {
		t.Error("CheckPassword failed for bcrypt-hashed password")
	}
}
