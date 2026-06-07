package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dolphinz/im-server/pkg/model"
)

// mockUserRepo is an in-memory implementation of userRepository for testing.
type mockUserRepo struct {
	users map[string]*model.User
	err   error // when non-nil, all operations return this error
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (m *mockUserRepo) Create(_ context.Context, u *model.User) error {
	if m.err != nil {
		return m.err
	}
	// Store a copy so that later mutations by the caller (e.g. clearing
	// Password) do not affect the stored record.
	cp := *u
	m.users[u.ID] = &cp
	return nil
}

func (m *mockUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	u, ok := m.users[id]
	if !ok {
		return nil, errors.New("user not found")
	}
	return u, nil
}

func (m *mockUserRepo) GetByAccount(_ context.Context, account string) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, u := range m.users {
		if u.Account == account {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

// setupService creates a Service backed by a fresh RSA key pair and an empty
// mock repository. It returns the service, the repository, and the jwt secret
// used so callers can sign tampered tokens.
func setupService(t *testing.T) (*Service, *mockUserRepo, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey: %v", err)
	}
	c := NewCryptoFromKeys(priv)
	repo := newMockUserRepo()
	jwtSecret := "test-jwt-secret"
	svc := NewService(c, jwtSecret, 24, repo)
	return svc, repo, jwtSecret
}

// ---------------------------------------------------------------------------
// Register
// ---------------------------------------------------------------------------

func TestRegister_Success(t *testing.T) {
	svc, repo, _ := setupService(t)
	ctx := context.Background()
	const (
		name     = "alice"
		password = "p@ssw0rd"
		account  = "alice_account"
	)

	user, token, err := svc.Register(ctx, name, password, account)
	if err != nil {
		t.Fatalf("Register returned unexpected error: %v", err)
	}

	// --- returned user fields ---
	if user.ID == "" {
		t.Error("expected non-empty ID")
		if user.Account != account {
			t.Errorf("Account = %q, want %q", user.Account, account)
		}
	}
	if user.Type != model.UserHuman {
		t.Errorf("Type = %d, want %d", user.Type, model.UserHuman)
	}
	if user.Name != name {
		t.Errorf("Name = %q, want %q", user.Name, name)
	}
	if user.Status != model.UserOffline {
		t.Errorf("Status = %d, want %d", user.Status, model.UserOffline)
	}
	if user.CreatedAt == 0 {
		t.Error("expected non-zero CreatedAt")
	}
	if user.Password != "" {
		t.Error("Password should be cleared on the returned user")
	}

	// --- stored user in repo ---
	stored, err := repo.GetByID(ctx, user.ID)
	if err != nil {
		t.Fatalf("stored user not found: %v", err)
	}
	if stored.Password == "" {
		t.Error("expected encrypted password in repo")
	}
	if stored.Password == password {
		t.Error("password must not be stored in plaintext")
	}

	// --- JWT ---
	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken(generated token): %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Type != int(model.UserHuman) {
		t.Errorf("claims.Type = %d, want %d", claims.Type, int(model.UserHuman))
	}
}

func TestRegister_RepoError(t *testing.T) {
	svc, repo, _ := setupService(t)
	ctx := context.Background()

	expected := errors.New("disk full")
	repo.err = expected
	t.Cleanup(func() { repo.err = nil })

	_, _, err := svc.Register(ctx, "bob", "secret", "bob_account")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expected) {
		t.Errorf("error does not wrap %v: %v", expected, err)
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestLogin_Success(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	user, _, err := svc.Register(ctx, "carol", "hunter2", "carol_account")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	token, expireAt, _, err := svc.Login(ctx, "carol_account", "hunter2")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// -- JWT claims --
	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken(login token): %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Type != int(model.UserHuman) {
		t.Errorf("claims.Type = %d, want %d", claims.Type, int(model.UserHuman))
	}

	// -- expire timestamp should be ~24 h in the future --
	now := time.Now().Unix()
	if expireAt <= now {
		t.Errorf("expireAt (%d) should be in the future (now=%d)", expireAt, now)
	}
	ttl := expireAt - now
	if ttl < 23*3600 || ttl > 25*3600 {
		t.Errorf("expireAt TTL = %d s, want roughly 86400 s (24 h)", ttl)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, err := svc.Register(ctx, "dave", "real-password", "dave_account")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, _, err = svc.Login(ctx, "dave_account", "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}

	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNoPermission {
		t.Errorf("AppError.Code = %d, want %d", appErr.Code, model.ErrNoPermission)
	}
}

func TestLogin_UserNotFound(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, _, err := svc.Login(ctx, "account_not_exist", "irrelevant")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
}

func TestLogin_WrongCryptoKey(t *testing.T) {
	// The user was registered with key-pair A, but the Login service uses
	// key-pair B, so RSA-OAEP decryption will fail.
	ctx := context.Background()

	// --- service with key A ---
	privA, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(A): %v", err)
	}
	cryptoA := NewCryptoFromKeys(privA)
	repo := newMockUserRepo()
	svcA := NewService(cryptoA, "shared-secret", 24, repo)

	_, _, err = svcA.Register(ctx, "eve", "secret", "eve_account")
	if err != nil {
		t.Fatalf("Register with key A: %v", err)
	}

	// --- service with key B (different) ---
	privB, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey(B): %v", err)
	}
	cryptoB := NewCryptoFromKeys(privB)
	svcB := NewService(cryptoB, "shared-secret", 24, repo)

	_, _, _, err = svcB.Login(ctx, "eve_account", "secret")
	if err == nil {
		t.Fatal("expected error when crypto keys differ, got nil")
	}
}

// ---------------------------------------------------------------------------
// ParseToken
// ---------------------------------------------------------------------------

func TestParseToken_Valid(t *testing.T) {
	svc, _, _ := setupService(t)

	token, err := svc.generateToken("user_42", int(model.UserHuman))
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if claims.UserID != "user_42" {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, "user_42")
	}
	if claims.Type != int(model.UserHuman) {
		t.Errorf("claims.Type = %d, want %d", claims.Type, int(model.UserHuman))
	}
	if claims.Issuer != "im-server" {
		t.Errorf("claims.Issuer = %q, want %q", claims.Issuer, "im-server")
	}
	if claims.IssuedAt == nil {
		t.Error("expected non-nil IssuedAt")
	}
	if claims.ExpiresAt == nil {
		t.Error("expected non-nil ExpiresAt")
	}
	if time.Now().After(claims.ExpiresAt.Time) {
		t.Error("ExpiresAt is in the past")
	}
}

func TestParseToken_Expired(t *testing.T) {
	svc, _, _ := setupService(t)

	claims := &Claims{
		UserID: "user_expired",
		Type:   int(model.UserHuman),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			Issuer:    "im-server",
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(svc.jwtSecret)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = svc.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParseToken_Tampered(t *testing.T) {
	svc, _, jwtSecret := setupService(t)

	claims := &Claims{
		UserID: "user_tampered",
		Type:   int(model.UserHuman),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "im-server",
		},
	}
	// Sign with a different secret
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(jwtSecret + "-tampered"))
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = svc.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestParseToken_WrongSigningMethod(t *testing.T) {
	svc, _, _ := setupService(t)

	claims := &Claims{
		UserID: "user_none",
		Type:   int(model.UserHuman),
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "im-server",
		},
	}
	// "None" algorithm – the service should reject it.
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("SignedString: %v", err)
	}

	_, err = svc.ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for None signing method, got nil")
	}
}
