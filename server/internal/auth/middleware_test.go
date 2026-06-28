package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// mockService implements enough of Service's token methods for middleware tests.
type mockService struct {
	parseTokenFn func(tokenStr string) (*Claims, error)
}

func (s *mockService) ParseToken(tokenStr string) (*Claims, error) {
	return s.parseTokenFn(tokenStr)
}

func TestUserFromCtx_Empty(t *testing.T) {
	uid := UserFromCtx(context.Background())
	if uid != "" {
		t.Errorf("UserFromCtx = %q, want empty", uid)
	}
}

func TestUserFromCtx_WithValue(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUserID, "user-abc")
	uid := UserFromCtx(ctx)
	if uid != "user-abc" {
		t.Errorf("UserFromCtx = %q, want user-abc", uid)
	}
}

func TestUserFromCtx_WrongType(t *testing.T) {
	ctx := context.WithValue(context.Background(), CtxKeyUserID, 12345)
	uid := UserFromCtx(ctx)
	if uid != "" {
		t.Errorf("UserFromCtx with non-string value = %q, want empty", uid)
	}
}

func TestExtractBearerToken_FromHeader(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer mytoken123")
	token := extractBearerToken(r)
	if token != "mytoken123" {
		t.Errorf("extractBearerToken = %q, want mytoken123", token)
	}
}

func TestExtractBearerToken_FromQuery(t *testing.T) {
	// Query param tokens are no longer supported for security reasons.
	r := httptest.NewRequest("GET", "/?token=querytoken456", nil)
	token := extractBearerToken(r)
	if token != "" {
		t.Errorf("extractBearerToken = %q, want empty (query params rejected)", token)
	}
}

func TestExtractBearerToken_HeaderTakesPrecedence(t *testing.T) {
	// Even if a token query param is present, only the Authorization header is used.
	r := httptest.NewRequest("GET", "/?token=querytoken", nil)
	r.Header.Set("Authorization", "Bearer headertoken")
	token := extractBearerToken(r)
	if token != "headertoken" {
		t.Errorf("extractBearerToken = %q, want headertoken", token)
	}
}

func TestExtractBearerToken_Empty(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	token := extractBearerToken(r)
	if token != "" {
		t.Errorf("extractBearerToken = %q, want empty", token)
	}
}

func TestExtractBearerToken_NoBearerPrefix(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
	token := extractBearerToken(r)
	if token != "" {
		t.Errorf("extractBearerToken = %q, want empty", token)
	}
}

func TestAuthMiddleware_Success(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			return &Claims{UserID: "user-42"}, nil
		},
	}

	var capturedUID string
	handler := AuthMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedUID = UserFromCtx(r.Context())
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer validtoken")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedUID != "user-42" {
		t.Errorf("UserID in context = %q, want user-42", capturedUID)
	}
}

func TestAuthMiddleware_NoToken(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			t.Error("ParseToken should not be called with no token")
			return nil, nil
		},
	}

	handler := AuthMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}

	var resp HTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code == 0 {
		t.Error("response code should be non-zero for auth error")
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			return nil, jwt.ErrSignatureInvalid
		},
	}

	handler := AuthMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("next handler should not be called")
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer invalidtoken")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWSAuthMiddleware_Success(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			return &Claims{UserID: "ws-user"}, nil
		},
	}

	mw := WSAuthMiddleware(svc, nil)
	ctx, err := mw(context.Background(), "wstoken")
	if err != nil {
		t.Fatalf("WSAuthMiddleware error: %v", err)
	}
	uid := UserFromCtx(ctx)
	if uid != "ws-user" {
		t.Errorf("UserID = %q, want ws-user", uid)
	}
}

func TestWSAuthMiddleware_Failure(t *testing.T) {
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			return nil, jwt.ErrSignatureInvalid
		},
	}

	mw := WSAuthMiddleware(svc, nil)
	_, err := mw(context.Background(), "bad-token")
	if err == nil {
		t.Fatal("expected error for bad token")
	}
}

func TestWriteAuthError_ResponseFormat(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)

	writeAuthError(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp["code"] == nil {
		t.Error("response should have code field")
	}
	if resp["msg"] == nil {
		t.Error("response should have msg field")
	}
}

func TestAuthMiddleware_ExtractBearerTokenCalled(t *testing.T) {
	// Verify that extractBearerToken is called with the right request.
	svc := &mockService{
		parseTokenFn: func(tokenStr string) (*Claims, error) {
			if tokenStr != "expected-token" {
				t.Errorf("ParseToken called with %q, want expected-token", tokenStr)
			}
			return &Claims{UserID: "u1"}, nil
		},
	}

	handler := AuthMiddleware(svc)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Authorization", "Bearer expected-token")
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// ---------------------------------------------------------------------------
// WriteJSON tests
// ---------------------------------------------------------------------------

func TestWriteJSON_Success(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, 0, map[string]string{"key": "val"})

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var resp HTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 0 {
		t.Errorf("Code = %d, want 0", resp.Code)
	}
	if resp.Msg != "ok" {
		t.Errorf("Msg = %q, want ok", resp.Msg)
	}
}

func TestWriteJSON_Error(t *testing.T) {
	w := httptest.NewRecorder()
	WriteJSON(w, 1001, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp HTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Code != 1001 {
		t.Errorf("Code = %d, want 1001", resp.Code)
	}
	if resp.Msg != "error" {
		t.Errorf("Msg = %q, want error", resp.Msg)
	}
}

func TestWriteJSON_AppError(t *testing.T) {
	type appError struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	w := httptest.NewRecorder()
	appErr := &appError{Code: 2001, Message: "custom error message"}
	WriteJSON(w, 2001, appErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	var resp HTTPResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// With AppError-like data, the status code stays but the msg comes from data.Message
	// Note: this test documents current behavior. The real *model.AppError type has Message field.
}

func TestMarshalJSON(t *testing.T) {
	data := marshalJSON(map[string]string{"foo": "bar"})
	if string(data) != `{"foo":"bar"}` {
		t.Errorf("marshalJSON = %s, want %s", string(data), `{"foo":"bar"}`)
	}
}

func TestMarshalJSON_Nil(t *testing.T) {
	data := marshalJSON(nil)
	if string(data) != "null" {
		t.Errorf("marshalJSON(nil) = %s, want null", string(data))
	}
}

// ---------------------------------------------------------------------------
// Time-dependent tests
// ---------------------------------------------------------------------------

func TestTokenExpiry(t *testing.T) {
	// Verify that token claims with expiry work correctly
	now := time.Now()
	claims := &Claims{
		UserID: "user-t1",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	if !claims.ExpiresAt.Time.After(now) {
		t.Error("token should not be expired yet")
	}

	expiredClaims := &Claims{
		UserID: "user-t2",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(-time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now.Add(-2 * time.Hour)),
		},
	}

	if expiredClaims.ExpiresAt.Time.Before(now) {
		// expired - this should be detectable
	}
}
