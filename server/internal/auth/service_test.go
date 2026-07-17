package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"ziziphus/pkg/model"
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

func (m *mockUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if m.err != nil {
		return nil, m.err
	}
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
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

// setupService creates a Service backed by a fresh mock repository.
func setupService(t *testing.T) (*Service, *mockUserRepo, string) {
	t.Helper()
	repo := newMockUserRepo()
	jwtSecret := "test-jwt-secret"
	svc := NewService(jwtSecret, 24, 168, repo, nil, func() int64 { return time.Now().UnixNano() }) // nil rdb disables refresh/blacklist
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

	user, accessToken, refreshToken, err := svc.Register(ctx, name, password, account, "", "")
	if err != nil {
		t.Fatalf("Register returned unexpected error: %v", err)
	}

	// --- returned user fields ---
	if user.ID == "" {
		t.Error("expected non-empty ID")
	}
	if user.Account != account {
		t.Errorf("Account = %q, want %q", user.Account, account)
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
		t.Error("expected hashed password in repo")
	}
	if stored.Password == password {
		t.Error("password must not be stored in plaintext")
	}
	// Verify it's a valid bcrypt hash
	if !CheckPassword(password, stored.Password) {
		t.Error("stored password does not match the original")
	}

	// --- JWT ---
	claims, err := svc.ParseToken(accessToken)
	if err != nil {
		t.Fatalf("ParseToken(generated token): %v", err)
	}
	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Type != int(model.UserHuman) {
		t.Errorf("claims.Type = %d, want %d", claims.Type, int(model.UserHuman))
	}

	// --- refresh token (nil rdb, so should be empty) ---
	if refreshToken != "" {
		t.Errorf("refreshToken = %q, want empty string (rdb is nil)", refreshToken)
	}
}

func TestRegister_DuplicateAccount(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, _, err := svc.Register(ctx, "alice", "pass1long", "same_account", "", "")
	if err != nil {
		t.Fatalf("first Register: %v", err)
	}

	_, _, _, err = svc.Register(ctx, "bob", "pass2long", "same_account", "", "")
	if err == nil {
		t.Fatal("expected error for duplicate account, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", appErr.Code, model.ErrBadMessage)
	}
}

func TestRegister_RepoError(t *testing.T) {
	svc, repo, _ := setupService(t)
	ctx := context.Background()

	expected := errors.New("disk full")
	repo.err = expected
	t.Cleanup(func() { repo.err = nil })

	_, _, _, err := svc.Register(ctx, "bob", "secret12", "bob_account", "", "")
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

	user, _, _, err := svc.Register(ctx, "carol", "hunter2!", "carol_account", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	accessToken, _, expireAt, _, err := svc.Login(ctx, "carol_account", "hunter2!")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	// -- JWT claims --
	claims, err := svc.ParseToken(accessToken)
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

	_, _, _, err := svc.Register(ctx, "dave", "real-password", "dave_account", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	_, _, _, _, err = svc.Login(ctx, "dave_account", "wrong-password")
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

	_, _, _, _, err := svc.Login(ctx, "account_not_exist", "irrelevant")
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}

	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	// Unified message: both "user not found" and "wrong password" return the same message
	// to prevent user enumeration.
	if appErr.Message != "invalid account or password" {
		t.Errorf("AppError.Message = %q; want %q (unified to prevent enumeration)", appErr.Message, "invalid account or password")
	}
}

// TestLogin_UnifiedErrorMessage verifies that Login returns the same error message
// for both wrong password and non-existent user, preventing account enumeration.
func TestLogin_UnifiedErrorMessage(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, _, err := svc.Register(ctx, "eve", "good-password", "eve_account", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Wrong password
	_, _, _, _, err1 := svc.Login(ctx, "eve_account", "bad-password")
	if err1 == nil {
		t.Fatal("expected error for wrong password")
	}
	var appErr1 *model.AppError
	if !errors.As(err1, &appErr1) {
		t.Fatalf("expected *model.AppError for wrong password, got %T", err1)
	}

	// Non-existent account
	_, _, _, _, err2 := svc.Login(ctx, "no_such_user", "irrelevant")
	if err2 == nil {
		t.Fatal("expected error for non-existent user")
	}
	var appErr2 *model.AppError
	if !errors.As(err2, &appErr2) {
		t.Fatalf("expected *model.AppError for missing user, got %T", err2)
	}

	// Both should return the same message — no enumeration possible.
	if appErr1.Message != appErr2.Message {
		t.Errorf("Messages differ: wrong password = %q, user not found = %q; must be identical to prevent enumeration",
			appErr1.Message, appErr2.Message)
	}
}

// TestRegister_PasswordMinLength verifies that passwords shorter than 8 characters are rejected.
func TestRegister_PasswordMinLength(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, _, err := svc.Register(ctx, "short", "1234567", "short_account", "", "")
	if err == nil {
		t.Fatal("expected error for password < 8 chars, got nil")
	}
}

// ---------------------------------------------------------------------------
// ParseToken
// ---------------------------------------------------------------------------

func TestParseToken_Valid(t *testing.T) {
	svc, _, _ := setupService(t)

	// generateAccessToken is unexported; use Register to get a real token
	user, token, _, err := svc.Register(context.Background(), "test", "testpass", "test_parse", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	claims, err := svc.ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken: %v", err)
	}

	if claims.UserID != user.ID {
		t.Errorf("claims.UserID = %q, want %q", claims.UserID, user.ID)
	}
	if claims.Type != int(model.UserHuman) {
		t.Errorf("claims.Type = %d, want %d", claims.Type, int(model.UserHuman))
	}
	if claims.Issuer != "ziziphus" {
		t.Errorf("claims.Issuer = %q, want %q", claims.Issuer, "ziziphus")
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
			Issuer:    "ziziphus",
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
			Issuer:    "ziziphus",
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
			Issuer:    "ziziphus",
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

// ---------------------------------------------------------------------------
// Password Reset
// ---------------------------------------------------------------------------

type mockPasswordResetStore struct {
	resets map[string]*model.PasswordReset
	err    error
}

func newMockPasswordResetStore() *mockPasswordResetStore {
	return &mockPasswordResetStore{resets: make(map[string]*model.PasswordReset)}
}

func (m *mockPasswordResetStore) Upsert(_ context.Context, pr *model.PasswordReset) error {
	if m.err != nil {
		return m.err
	}
	cp := *pr
	m.resets[pr.UserID] = &cp
	return nil
}

func (m *mockPasswordResetStore) Get(_ context.Context, userID string) (*model.PasswordReset, error) {
	if m.err != nil {
		return nil, m.err
	}
	pr, ok := m.resets[userID]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *pr
	return &cp, nil
}

func (m *mockPasswordResetStore) Delete(_ context.Context, userID string) error {
	delete(m.resets, userID)
	return m.err
}

type mockPasswordUpdater struct {
	err error
}

func (m *mockPasswordUpdater) UpdatePassword(_ context.Context, userID, password string) error {
	return m.err
}

type mockPasswordResetMailer struct {
	err error
}

func (m *mockPasswordResetMailer) Enabled() bool {
	return m.err == nil
}

func (m *mockPasswordResetMailer) SendPasswordResetCode(_, _ string) error {
	return m.err
}

func TestRequestPasswordReset_Success(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	// Register a user with email
	user, _, _, err := svc.Register(ctx, "alice", "password123", "alice_reset", "alice@example.com", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	mailer := &mockPasswordResetMailer{}

	userID, code, err := svc.RequestPasswordReset(ctx, "alice_reset", resetStore, mailer, true)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}
	if userID != user.ID {
		t.Errorf("userID = %q, want %q", userID, user.ID)
	}
	if code == "" {
		t.Error("expected non-empty reset code")
	}

	// Verify the code was stored
	stored, err := resetStore.Get(ctx, userID)
	if err != nil {
		t.Fatalf("Get stored reset: %v", err)
	}
	if stored.Code != code {
		t.Errorf("stored code = %q, want %q", stored.Code, code)
	}
	if stored.ExpiresAt.Before(time.Now()) {
		t.Error("stored ExpiresAt is in the past")
	}
}

func TestRequestPasswordReset_ByEmail(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	// Register a user with email
	user, _, _, err := svc.Register(ctx, "bob", "password123", "bob_reset_email", "bob@example.com", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	mailer := &mockPasswordResetMailer{}

	// Look up by email instead of account
	userID, code, err := svc.RequestPasswordReset(ctx, "bob@example.com", resetStore, mailer, true)
	if err != nil {
		t.Fatalf("RequestPasswordReset by email: %v", err)
	}
	if userID != user.ID {
		t.Errorf("userID = %q, want %q", userID, user.ID)
	}
	if code == "" {
		t.Error("expected non-empty reset code")
	}
}

func TestRequestPasswordReset_NoEmail(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	// Register a user without email
	_, _, _, err := svc.Register(ctx, "charlie", "password123", "charlie_noemail", "", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	mailer := &mockPasswordResetMailer{}

	_, _, err = svc.RequestPasswordReset(ctx, "charlie_noemail", resetStore, mailer, false)
	if err == nil {
		t.Fatal("expected error for user without email, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", appErr.Code, model.ErrBadMessage)
	}
}

func TestRequestPasswordReset_UserNotFound(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	resetStore := newMockPasswordResetStore()
	mailer := &mockPasswordResetMailer{}

	_, _, err := svc.RequestPasswordReset(ctx, "nonexistent_user", resetStore, mailer, false)
	if err == nil {
		t.Fatal("expected error for non-existent user, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
}

func TestRequestPasswordReset_NotFoundByEmail(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	resetStore := newMockPasswordResetStore()
	mailer := &mockPasswordResetMailer{}

	_, _, err := svc.RequestPasswordReset(ctx, "nonexistent@example.com", resetStore, mailer, false)
	if err == nil {
		t.Fatal("expected error for non-existent email, got nil")
	}
}

func TestResetPassword_Success(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	user, _, _, err := svc.Register(ctx, "dave", "oldpassword", "dave_reset", "dave@example.com", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	pwUpdater := &mockPasswordUpdater{}
	mailer := &mockPasswordResetMailer{}
	newPassword := "newpassword456"

	// Request reset
	_, code, err := svc.RequestPasswordReset(ctx, "dave_reset", resetStore, mailer, true)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	// Reset password
	if err := svc.ResetPassword(ctx, user.ID, code, newPassword, resetStore, pwUpdater); err != nil {
		t.Fatalf("ResetPassword: %v", err)
	}

	// Verify the code was deleted after use
	_, err = resetStore.Get(ctx, user.ID)
	if err == nil {
		t.Error("expected reset code to be deleted after use")
	}
}

func TestResetPassword_WrongCode(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	user, _, _, err := svc.Register(ctx, "eve", "oldpassword", "eve_wrong", "eve@example.com", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	pwUpdater := &mockPasswordUpdater{}
	mailer := &mockPasswordResetMailer{}

	// Request reset
	_, code, err := svc.RequestPasswordReset(ctx, "eve_wrong", resetStore, mailer, true)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	// Try to reset with wrong code
	wrongCode := "000000"
	if wrongCode == code {
		wrongCode = "000001"
	}
	err = svc.ResetPassword(ctx, user.ID, wrongCode, "newpassword456", resetStore, pwUpdater)
	if err == nil {
		t.Fatal("expected error for wrong code, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
}

func TestResetPassword_ExpiredCode(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	user, _, _, err := svc.Register(ctx, "frank", "oldpassword", "frank_exp", "frank@example.com", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	resetStore := newMockPasswordResetStore()
	pwUpdater := &mockPasswordUpdater{}
	mailer := &mockPasswordResetMailer{}

	// Request reset and store the code
	_, code, err := svc.RequestPasswordReset(ctx, "frank_exp", resetStore, mailer, true)
	if err != nil {
		t.Fatalf("RequestPasswordReset: %v", err)
	}

	// Manually expire the code
	stored, _ := resetStore.Get(ctx, user.ID)
	stored.ExpiresAt = time.Now().Add(-1 * time.Minute)
	resetStore.resets[user.ID] = stored

	err = svc.ResetPassword(ctx, user.ID, code, "newpassword456", resetStore, pwUpdater)
	if err == nil {
		t.Fatal("expected error for expired code, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
}

func TestResetPassword_ShortPassword(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	resetStore := newMockPasswordResetStore()
	pwUpdater := &mockPasswordUpdater{}

	err := svc.ResetPassword(ctx, "any_user", "123456", "short", resetStore, pwUpdater)
	if err == nil {
		t.Fatal("expected error for short password, got nil")
	}
}
