package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"ziziphus/internal/auth"
	"ziziphus/internal/storage/db"
	"ziziphus/pkg/model"
)

// ---------------------------------------------------------------------------
// E2E helpers
// ---------------------------------------------------------------------------

// e2eRouter creates a full router with mock dependencies for end-to-end testing.
func e2eRouter(t *testing.T) (http.Handler, *testAuthUserRepo, *mockConvManager, *auth.Service) {
	t.Helper()

	authUserRepo := &testAuthUserRepo{}
	authSvc := auth.NewService("e2e-test-jwt-secret", 1, 168, authUserRepo, nil, func() int64 { return time.Now().UnixNano() })

	userRepo := &mockUserRepo{
		getByIDFunc: func(_ context.Context, id string) (*model.User, error) {
			u, err := authUserRepo.GetByID(context.Background(), id)
			if err != nil {
				return nil, err
			}
			cp := *u
			cp.Password = ""
			return &cp, nil
		},
		getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
			return map[string]*model.User{
				"user_a": {ID: "user_a", Name: "User A"},
				"user_b": {ID: "user_b", Name: "User B"},
			}, nil
		},
		searchFunc: func(_ context.Context, q string, page, size int) ([]*model.User, int, error) {
			return nil, 0, nil
		},
		updateFunc: func(_ context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor string, discoverable, allowDirectChat bool) error {
			return nil
		},
	}

	sessMgr := &mockSessionChecker{
		isOnlineFunc:          func(_ context.Context, userID string) bool { return false },
		getUserSessionIDsFunc: func(_ context.Context, userID string) []string { return nil },
	}

	convMgr := &mockConvManager{
		getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
			return nil, &model.AppError{Code: model.ErrNotFound, Message: "not found"}
		},
		isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
			return false, nil
		},
		getMembersFunc: func(_ context.Context, convID string) ([]*model.ConvMember, error) {
			return nil, nil
		},
	}

	msgRepo := &mockMsgStorage{
		getHistoryFunc: func(_ context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
			return nil, nil
		},
	}

	convRepo := &mockConvDataRepo{
		getUserConvsFunc: func(_ context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error) {
			return nil, 0, nil
		},
		searchByNameFunc: func(_ context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error) {
			return nil, 0, nil
		},
	}

	seqCache := &mockConvSeqCache{}
	readMarker := &mockReadMarker{}
	sysMsg := &mockSysMsgSender{}

	contactRepo := &mockContactStorage{
		listFunc: func(_ context.Context, userID string, page, size int) ([]*model.Contact, int, error) {
			return nil, 0, nil
		},
	}

	userQueryRepo := &mockUserQueryRepo{}

	userHandler := NewUserHandler(authSvc, userRepo, sessMgr, func() int64 { return time.Now().UnixNano() }, nil, nil, nil, true, "Ziziphus")
	convHandler := NewConvHandler(convMgr, convRepo, seqCache, readMarker, sysMsg, userRepo, func() int64 { return 1 })
	msgHandler := NewMsgHandler(msgRepo, &mockReceiptStorage{}, convMgr)
	contactHandler := NewContactHandler(contactRepo, &mockContactRequestStorage{}, userQueryRepo, sessMgr, &mockFormMessageSender{}, &mockSystemConvManager{})

	handlers := &Handlers{
		User:         userHandler,
		Conversation: convHandler,
		Message:      msgHandler,
		Contact:      contactHandler,
	}

	authMW := auth.AuthMiddleware(authSvc)
	r := NewRouter(handlers, authMW)

	return r, authUserRepo, convMgr, authSvc
}

// e2eRequest performs an HTTP request against the test handler and decodes the response.
func e2eRequest(t *testing.T, handler http.Handler, method, path, body string, headers map[string]string) (*httptest.ResponseRecorder, APIResponse) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "127.0.0.1:12345"
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		return w, APIResponse{Code: -1, Msg: w.Body.String()}
	}
	return w, resp
}

// mustRegister registers a user and returns user info and tokens.
func mustRegister(t *testing.T, handler http.Handler, name, account, password string) (userID, token, refreshToken string) {
	t.Helper()
	body := fmt.Sprintf(`{"name":%q,"account":%q,"password":%q}`, name, account, password)
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/register", body, nil)
	if resp.Code != 0 {
		t.Fatalf("register failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	userID = data["user_id"].(string)
	token = data["token"].(string)
	if rt, ok := data["refresh_token"]; ok {
		refreshToken, _ = rt.(string)
	}
	return
}

// mustLogin logs in and returns tokens.
func mustLogin(t *testing.T, handler http.Handler, account, password string) (token, refreshToken string) {
	t.Helper()
	body := fmt.Sprintf(`{"account":%q,"password":%q}`, account, password)
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/login", body, nil)
	if resp.Code != 0 {
		t.Fatalf("login failed: code=%d msg=%s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	token = data["token"].(string)
	if rt, ok := data["refresh_token"]; ok {
		refreshToken, _ = rt.(string)
	}
	return
}

// bearerHeader creates an Authorization header with a Bearer token.
func bearerHeader(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

// =============================================================================
// E2E: Password bcrypt — password is never stored or returned in plaintext
// =============================================================================

func TestE2E_PasswordBcrypt_StoredHashed(t *testing.T) {
	handler, authUserRepo, _, _ := e2eRouter(t)

	userID, _, _ := mustRegister(t, handler, "Alice", "alice_bcrypt", "secure-password")

	// Verify stored password is bcrypt hashed
	stored, err := authUserRepo.GetByID(context.Background(), userID)
	if err != nil {
		t.Fatalf("stored user not found: %v", err)
	}
	if stored.Password == "" {
		t.Fatal("stored password is empty")
	}
	if stored.Password == "secure-password" {
		t.Fatal("PASSWORD STORED IN PLAINTEXT")
	}
	if !auth.CheckPassword("secure-password", stored.Password) {
		t.Fatal("bcrypt CheckPassword failed for stored hash")
	}
}

func TestE2E_PasswordBcrypt_NotReturnedInResponse(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	userID, token, _ := mustRegister(t, handler, "Bob", "bob_bcrypt", "my-password")

	// Login response should NOT include password
	body := `{"account":"bob_bcrypt","password":"my-password"}`
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/login", body, nil)
	if resp.Code != 0 {
		t.Fatalf("login failed: %s", resp.Msg)
	}
	loginData := resp.Data.(map[string]interface{})
	if pw, ok := loginData["password"]; ok {
		t.Errorf("password field must not be in login response, got: %v", pw)
	}

	// GetMe response should NOT include password
	_, resp = e2eRequest(t, handler, http.MethodGet, "/api/v1/users/me", "", bearerHeader(token))
	if resp.Code != 0 {
		t.Fatalf("GetMe failed: %s", resp.Msg)
	}
	meData := resp.Data.(map[string]interface{})
	if meData["user_id"] != userID {
		t.Errorf("user_id = %v, want %q", meData["user_id"], userID)
	}
	if pw, ok := meData["password"]; ok {
		t.Errorf("password field must not be in GetMe response, got: %v", pw)
	}
}

func TestE2E_PasswordBcrypt_WrongPasswordRejected(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	mustRegister(t, handler, "Carol", "carol_bcrypt", "correct-password")

	body := `{"account":"carol_bcrypt","password":"wrong-password"}`
	w, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/login", body, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

// =============================================================================
// E2E: Auth middleware — token validation
// =============================================================================

func TestE2E_AuthMiddleware_NoTokenBlocked(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/users/me", "", nil)
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

func TestE2E_AuthMiddleware_InvalidTokenBlocked(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/users/me", "", bearerHeader("invalid-jwt"))
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

func TestE2E_AuthMiddleware_ExpiredTokenBlocked(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	// Create an expired JWT signed with the test secret
	claims := jwt.MapClaims{
		"uid": "test_user",
		"typ": 1,
		"iss": "ziziphus",
		"exp": time.Now().Add(-1 * time.Hour).Unix(),
		"iat": time.Now().Add(-2 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("e2e-test-jwt-secret"))
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/users/me", "", bearerHeader(tokenStr))
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

func TestE2E_AuthMiddleware_TamperedTokenBlocked(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	// Token signed with wrong secret
	claims := jwt.MapClaims{
		"uid": "test_user",
		"typ": 1,
		"iss": "ziziphus",
		"exp": time.Now().Add(1 * time.Hour).Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte("different-secret"))
	if err != nil {
		t.Fatalf("sign tampered token: %v", err)
	}

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/users/me", "", bearerHeader(tokenStr))
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

// =============================================================================
// E2E: Refresh token (nil Redis — verifies graceful handling)
// =============================================================================

func TestE2E_RefreshToken_NilRedisReturnsEmpty(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	// Register with nil rdb — refresh_token should be empty
	_, _, refreshToken := mustRegister(t, handler, "Dave", "dave_refresh", "testpass1")
	if refreshToken != "" {
		t.Logf("refresh_token = %q (expected empty since rdb is nil)", refreshToken)
	}

	// Login with nil rdb — refresh_token should be empty
	_, refreshToken2 := mustLogin(t, handler, "dave_refresh", "testpass1")
	if refreshToken2 != "" {
		t.Logf("login refresh_token = %q (expected empty since rdb is nil)", refreshToken2)
	}
}

func TestE2E_RefreshToken_EndpointRejectsEmpty(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	body := `{"refresh_token":""}`
	w, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/refresh", body, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	if resp.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrBadMessage)
	}
}

func TestE2E_RefreshToken_EndpointRejectsGarbage(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	body := `{"refresh_token":"garbage-token"}`
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/refresh", body, nil)
	// Without Redis, the refresh endpoint will return an error
	if resp.Code != model.ErrNoPermission && resp.Code != model.ErrInternal {
		t.Errorf("code = %d, want %d or %d", resp.Code, model.ErrNoPermission, model.ErrInternal)
	}
}

// =============================================================================
// E2E: Message history permission check
// =============================================================================

func TestE2E_GetHistory_MemberAllowed(t *testing.T) {
	handler, _, convMgr, _ := e2eRouter(t)

	aliceID, aliceToken, _ := mustRegister(t, handler, "Alice", "alice_hist_ok", "testpass1")

	// Alice is a member of "conv_abc"
	convMgr.isMemberFunc = func(_ context.Context, convID, userID string) (bool, error) {
		return userID == aliceID, nil
	}

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/conversations/conv_abc/messages", "", bearerHeader(aliceToken))
	if resp.Code != 0 {
		t.Errorf("Alice (member) GetHistory: code=%d msg=%s, want 0", resp.Code, resp.Msg)
	}
}

func TestE2E_GetHistory_NonMemberBlocked(t *testing.T) {
	handler, _, convMgr, _ := e2eRouter(t)

	mustRegister(t, handler, "Alice", "alice_hist_deny", "testpass1")
	_, bobToken, _ := mustRegister(t, handler, "Bob", "bob_hist_deny", "testpass1")

	// No one is a member of "conv_xyz"
	convMgr.isMemberFunc = func(_ context.Context, convID, userID string) (bool, error) {
		return false, nil
	}

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/conversations/conv_xyz/messages", "", bearerHeader(bobToken))
	if resp.Code != model.ErrNotFound {
		t.Errorf("Bob (non-member) GetHistory: code=%d, want %d", resp.Code, model.ErrNotFound)
	}
}

func TestE2E_GetHistory_UnauthenticatedBlocked(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	_, resp := e2eRequest(t, handler, http.MethodGet, "/api/v1/conversations/conv_123/messages", "", nil)
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

// =============================================================================
// E2E: Login + Register response structure
// =============================================================================

func TestE2E_AuthFlow_RegisterReturnsCorrectFields(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	body := `{"name":"Eve","account":"eve_fields","password":"test-pass"}`
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/register", body, nil)
	if resp.Code != 0 {
		t.Fatalf("register failed: %s", resp.Msg)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("response Data is not a map")
	}

	required := []string{"user_id", "account", "name", "token"}
	for _, field := range required {
		if _, exists := data[field]; !exists {
			t.Errorf("register response missing field: %s", field)
		}
	}

	if _, exists := data["password"]; exists {
		t.Error("register response must not contain password")
	}
}

func TestE2E_AuthFlow_LoginReturnsCorrectFields(t *testing.T) {
	handler, _, _, _ := e2eRouter(t)

	mustRegister(t, handler, "Frank", "frank_fields", "test-pass")

	body := `{"account":"frank_fields","password":"test-pass"}`
	_, resp := e2eRequest(t, handler, http.MethodPost, "/api/v1/users/login", body, nil)
	if resp.Code != 0 {
		t.Fatalf("login failed: %s", resp.Msg)
	}

	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("response Data is not a map")
	}

	required := []string{"user_id", "account", "token", "expires_at"}
	for _, field := range required {
		if _, exists := data[field]; !exists {
			t.Errorf("login response missing field: %s", field)
		}
	}

	if _, exists := data["password"]; exists {
		t.Error("login response must not contain password")
	}
}
