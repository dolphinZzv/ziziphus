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

	"github.com/go-chi/chi/v5"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/storage/db"
	"siciv.space/agent/panda_ai/pkg/model"
)

// ---------------------------------------------------------------------------
// testAuthUserRepo — satisfies auth.userRepository (unexported interface)
// ---------------------------------------------------------------------------

type testAuthUserRepo struct {
	users map[string]*model.User
}

func (r *testAuthUserRepo) Create(_ context.Context, u *model.User) error {
	if r.users == nil {
		r.users = make(map[string]*model.User)
	}
	// Make a copy so the caller can mutate the original without affecting the stored copy.
	cp := *u
	r.users[u.ID] = &cp
	return nil
}

func (r *testAuthUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	if r.users == nil {
		return nil, fmt.Errorf("user not found")
	}
	u, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found")
	}
	return u, nil
}

func (r *testAuthUserRepo) GetByAccount(_ context.Context, account string) (*model.User, error) {
	if r.users == nil {
		return nil, fmt.Errorf("user not found")
	}
	for _, u := range r.users {
		if u.Account == account {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

// ---------------------------------------------------------------------------
// Mock: userRepo (for UserHandler)
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	createFunc   func(ctx context.Context, u *model.User) error
	getByIDFunc  func(ctx context.Context, id string) (*model.User, error)
	getByIDsFunc func(ctx context.Context, ids []string) (map[string]*model.User, error)
	searchFunc   func(ctx context.Context, q string, page, size int) ([]*model.User, int, error)
	updateFunc   func(ctx context.Context, id, name, avatar, primaryColor, secondaryColor string) error
}

func (m *mockUserRepo) Create(ctx context.Context, u *model.User) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, u)
	}
	return nil
}

func (m *mockUserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockUserRepo) GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error) {
	if m.getByIDsFunc != nil {
		return m.getByIDsFunc(ctx, ids)
	}
	return nil, nil
}

func (m *mockUserRepo) Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error) {
	if m.searchFunc != nil {
		return m.searchFunc(ctx, q, page, size)
	}
	return nil, 0, nil
}

func (m *mockUserRepo) Update(ctx context.Context, id, name, avatar, primaryColor, secondaryColor string) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, avatar, primaryColor, secondaryColor)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: sessionChecker
// ---------------------------------------------------------------------------

type mockSessionChecker struct {
	isOnlineFunc          func(ctx context.Context, userID string) bool
	getUserSessionIDsFunc func(ctx context.Context, userID string) []string
}

func (m *mockSessionChecker) IsOnline(ctx context.Context, userID string) bool {
	if m.isOnlineFunc != nil {
		return m.isOnlineFunc(ctx, userID)
	}
	return false
}

func (m *mockSessionChecker) GetUserSessionIDs(ctx context.Context, userID string) []string {
	if m.getUserSessionIDsFunc != nil {
		return m.getUserSessionIDsFunc(ctx, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: convManager
// ---------------------------------------------------------------------------

type mockConvManager struct {
	getFunc                func(ctx context.Context, convID string) (*model.Conversation, error)
	createGroupFunc        func(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error)
	getOrCreateP2PFunc     func(ctx context.Context, userA, userB string) (*model.Conversation, error)
	addMemberFunc          func(ctx context.Context, convID, userID, operatorID string) error
	removeMemberFunc       func(ctx context.Context, convID, userID, operatorID string) error
	leaveFunc              func(ctx context.Context, convID, userID string) error
	getMembersFunc         func(ctx context.Context, convID string) ([]*model.ConvMember, error)
	isMemberFunc           func(ctx context.Context, convID, userID string) (bool, error)
	requestJoinFunc        func(ctx context.Context, convID, userID string) error
	listJoinRequestsFunc   func(ctx context.Context, convID, operatorID string) ([]*model.JoinRequest, error)
	approveJoinRequestFunc func(ctx context.Context, convID, userID, operatorID string) error
	rejectJoinRequestFunc  func(ctx context.Context, convID, userID, operatorID string) error
	getMemberRoleFunc      func(ctx context.Context, convID, userID string) (model.ConvRole, error)
}

func (m *mockConvManager) Get(ctx context.Context, convID string) (*model.Conversation, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, convID)
	}
	return nil, &model.AppError{Code: model.ErrNotFound, Message: "not found"}
}

func (m *mockConvManager) CreateGroup(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
	if m.createGroupFunc != nil {
		return m.createGroupFunc(ctx, name, ownerID, memberIDs, idGen)
	}
	return nil, nil
}

func (m *mockConvManager) GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error) {
	if m.getOrCreateP2PFunc != nil {
		return m.getOrCreateP2PFunc(ctx, userA, userB)
	}
	return nil, nil
}

func (m *mockConvManager) AddMember(ctx context.Context, convID, userID, operatorID string) error {
	if m.addMemberFunc != nil {
		return m.addMemberFunc(ctx, convID, userID, operatorID)
	}
	return nil
}

func (m *mockConvManager) RemoveMember(ctx context.Context, convID, userID, operatorID string) error {
	if m.removeMemberFunc != nil {
		return m.removeMemberFunc(ctx, convID, userID, operatorID)
	}
	return nil
}

func (m *mockConvManager) Leave(ctx context.Context, convID, userID string) error {
	if m.leaveFunc != nil {
		return m.leaveFunc(ctx, convID, userID)
	}
	return nil
}

func (m *mockConvManager) GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error) {
	if m.getMembersFunc != nil {
		return m.getMembersFunc(ctx, convID)
	}
	return nil, nil
}

func (m *mockConvManager) IsMember(ctx context.Context, convID, userID string) (bool, error) {
	if m.isMemberFunc != nil {
		return m.isMemberFunc(ctx, convID, userID)
	}
	return false, nil
}

func (m *mockConvManager) RequestJoin(ctx context.Context, convID, userID string) error {
	if m.requestJoinFunc != nil {
		return m.requestJoinFunc(ctx, convID, userID)
	}
	return nil
}

func (m *mockConvManager) ListJoinRequests(ctx context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
	if m.listJoinRequestsFunc != nil {
		return m.listJoinRequestsFunc(ctx, convID, operatorID)
	}
	return nil, nil
}

func (m *mockConvManager) ApproveJoinRequest(ctx context.Context, convID, userID, operatorID string) error {
	if m.approveJoinRequestFunc != nil {
		return m.approveJoinRequestFunc(ctx, convID, userID, operatorID)
	}
	return nil
}

func (m *mockConvManager) RejectJoinRequest(ctx context.Context, convID, userID, operatorID string) error {
	if m.rejectJoinRequestFunc != nil {
		return m.rejectJoinRequestFunc(ctx, convID, userID, operatorID)
	}
	return nil
}

func (m *mockConvManager) GetMemberRole(ctx context.Context, convID, userID string) (model.ConvRole, error) {
	if m.getMemberRoleFunc != nil {
		return m.getMemberRoleFunc(ctx, convID, userID)
	}
	return model.ConvRoleMember, nil
}

// ---------------------------------------------------------------------------
// Mock: convSeqCache
// ---------------------------------------------------------------------------

type mockConvSeqCache struct {
	getUnreadCountFunc func(ctx context.Context, userID, convID string) (int64, error)
}

func (m *mockConvSeqCache) GetUnreadCount(ctx context.Context, userID, convID string) (int64, error) {
	if m.getUnreadCountFunc != nil {
		return m.getUnreadCountFunc(ctx, userID, convID)
	}
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock: readMarker
// ---------------------------------------------------------------------------

type mockReadMarker struct {
	markReadFunc func(ctx context.Context, userID, convID string, msgID int64) error
}

func (m *mockReadMarker) MarkRead(ctx context.Context, userID, convID string, msgID int64) error {
	if m.markReadFunc != nil {
		return m.markReadFunc(ctx, userID, convID, msgID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: sysMsgSender
// ---------------------------------------------------------------------------

type mockSysMsgSender struct {
	sendSystemMessageFunc func(ctx context.Context, convID, body string) (*model.Message, error)
}

func (m *mockSysMsgSender) SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error) {
	if m.sendSystemMessageFunc != nil {
		return m.sendSystemMessageFunc(ctx, convID, body)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: convDataRepo
// ---------------------------------------------------------------------------

type mockConvDataRepo struct {
	getUserConvsFunc     func(ctx context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error)
	updateNameAvatarFunc func(ctx context.Context, convID, name, avatar string) error
	updateNoticeFunc     func(ctx context.Context, convID, notice string) error
	updateCoverFunc      func(ctx context.Context, convID, cover string) error
	searchByNameFunc     func(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error)
}

func (m *mockConvDataRepo) GetUserConvs(ctx context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error) {
	if m.getUserConvsFunc != nil {
		return m.getUserConvsFunc(ctx, userID, page, size)
	}
	return nil, 0, nil
}

func (m *mockConvDataRepo) UpdateNameAvatar(ctx context.Context, convID, name, avatar string) error {
	if m.updateNameAvatarFunc != nil {
		return m.updateNameAvatarFunc(ctx, convID, name, avatar)
	}
	return nil
}

func (m *mockConvDataRepo) UpdateNotice(ctx context.Context, convID, notice string) error {
	if m.updateNoticeFunc != nil { return m.updateNoticeFunc(ctx, convID, notice) }
	return nil
}

func (m *mockConvDataRepo) UpdateCover(ctx context.Context, convID, cover string) error {
	if m.updateCoverFunc != nil { return m.updateCoverFunc(ctx, convID, cover) }
	return nil
}

func (m *mockConvDataRepo) Pin(ctx context.Context, userID, convID string) error { return nil }
func (m *mockConvDataRepo) Unpin(ctx context.Context, userID, convID string) error { return nil }
func (m *mockConvDataRepo) Clone(ctx context.Context, src, dst, owner, name string, idGen func() int64) error { return nil }

func (m *mockConvDataRepo) AreContacts(ctx context.Context, userA, userB string) (bool, error) { return false, nil }

func (m *mockConvDataRepo) SearchByName(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error) {
	if m.searchByNameFunc != nil {
		return m.searchByNameFunc(ctx, q, page, size)
	}
	return nil, 0, nil
}

// ---------------------------------------------------------------------------
// Mock: msgStorage
// ---------------------------------------------------------------------------

type mockMsgStorage struct {
	getHistoryFunc func(ctx context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error)
}

func (m *mockMsgStorage) GetHistory(ctx context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
	if m.getHistoryFunc != nil {
		return m.getHistoryFunc(ctx, convID, beforeMsgID, aroundMsgID, limit, keyword, startDate, endDate)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: contactStorage
// ---------------------------------------------------------------------------

type mockContactStorage struct {
	addFunc            func(ctx context.Context, c *model.Contact) error
	removeFunc         func(ctx context.Context, userID, contactID string) error
	listFunc           func(ctx context.Context, userID string, page, size int) ([]*model.Contact, int, error)
	updateNicknameFunc func(ctx context.Context, userID, contactID, nickname string) error
}

func (m *mockContactStorage) Add(ctx context.Context, c *model.Contact) error {
	if m.addFunc != nil {
		return m.addFunc(ctx, c)
	}
	return nil
}

func (m *mockContactStorage) Remove(ctx context.Context, userID, contactID string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, userID, contactID)
	}
	return nil
}

func (m *mockContactStorage) List(ctx context.Context, userID string, page, size int) ([]*model.Contact, int, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx, userID, page, size)
	}
	return nil, 0, nil
}

func (m *mockContactStorage) UpdateNickname(ctx context.Context, userID, contactID, nickname string) error {
	if m.updateNicknameFunc != nil {
		return m.updateNicknameFunc(ctx, userID, contactID, nickname)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: userQueryRepo (for ContactHandler)
// ---------------------------------------------------------------------------

type mockUserQueryRepo struct {
	getByIDsFunc func(ctx context.Context, ids []string) (map[string]*model.User, error)
}

func (m *mockUserQueryRepo) GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error) {
	if m.getByIDsFunc != nil {
		return m.getByIDsFunc(ctx, ids)
	}
	return nil, nil
}

type mockContactRequestStorage struct{}

func (m *mockContactRequestStorage) Insert(_ context.Context, _ *model.ContactRequest) (int64, error) { return 1, nil }
func (m *mockContactRequestStorage) UpdateFormMsgID(_ context.Context, _, _ int64) error { return nil }
func (m *mockContactRequestStorage) GetByID(_ context.Context, _ int64) (*model.ContactRequest, error) { return nil, nil }
func (m *mockContactRequestStorage) GetByFormMsgID(_ context.Context, _ int64) (*model.ContactRequest, error) { return nil, nil }
func (m *mockContactRequestStorage) GetByPair(_ context.Context, _, _ string) (*model.ContactRequest, error) { return nil, nil }
func (m *mockContactRequestStorage) ListSent(_ context.Context, _ string, _, _ int) ([]*model.ContactRequest, int, error) { return nil, 0, nil }
func (m *mockContactRequestStorage) ListReceived(_ context.Context, _ string, _ int, _, _ int) ([]*model.ContactRequest, int, error) { return nil, 0, nil }
func (m *mockContactRequestStorage) Delete(_ context.Context, _ int64) error { return nil }
func (m *mockContactRequestStorage) ExistsAnyDirection(_ context.Context, _, _ string) (bool, error) { return false, nil }

type mockFormMessageSender struct{}

func (m *mockFormMessageSender) SendFormMessage(_ context.Context, _ string, _ *model.FormDefinitionBody) (*model.Message, error) {
	return &model.Message{MsgID: 1}, nil
}
func (m *mockFormMessageSender) SendSystemMessage(_ context.Context, _ string, _ string, _ ...string) (*model.Message, error) {
	return &model.Message{MsgID: 2}, nil
}

type mockSystemConvManager struct{}

func (m *mockSystemConvManager) GetOrCreateSystemConv(_ context.Context, _ string) (*model.Conversation, error) {
	return &model.Conversation{ConvID: "sys:test"}, nil
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setChiURLParam(r *http.Request, key, value string) *http.Request {
	// Preserve any URL params already set on the context (e.g. when multiple
	// params are needed on the same request).
	chiCtx, ok := r.Context().Value(chi.RouteCtxKey).(*chi.Context)
	if !ok {
		chiCtx = chi.NewRouteContext()
	}
	chiCtx.URLParams.Add(key, value)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
}

func setAuthCtx(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUserID, userID))
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// =========================================================================
// Tests: Response helpers
// =========================================================================

func TestJSON_writesCorrectContentTypeAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	JSON(w, map[string]string{"key": "value"})

	resp := w.Result()
	resp.Body.Close()

	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := decodeResponse(t, w)
	if body.Code != 0 {
		t.Errorf("code = %d, want 0", body.Code)
	}
	if body.Msg != "ok" {
		t.Errorf("msg = %q, want %q", body.Msg, "ok")
	}
}

func TestError_writesCorrectStatusAndBody(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	appErr := &model.AppError{Code: model.ErrBadMessage, Message: "test error"}
	Error(w, req, http.StatusBadRequest, appErr)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeResponse(t, w)
	if body.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", body.Code, model.ErrBadMessage)
	}
	if body.Msg != "test error" {
		t.Errorf("msg = %q, want %q", body.Msg, "test error")
	}
}

func TestPaginated_writesPaginatedStructure(t *testing.T) {
	w := httptest.NewRecorder()
	items := []string{"a", "b"}
	Paginated(w, items, 10, 1, 20)

	body := decodeResponse(t, w)
	if body.Code != 0 {
		t.Fatalf("code = %d, want 0", body.Code)
	}
	data, ok := body.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["total"].(float64) != 10 {
		t.Errorf("total = %v, want 10", data["total"])
	}
	if data["page"].(float64) != 1 {
		t.Errorf("page = %v, want 1", data["page"])
	}
	if data["size"].(float64) != 20 {
		t.Errorf("size = %v, want 20", data["size"])
	}
	itms, ok := data["items"].([]interface{})
	if !ok || len(itms) != 2 {
		t.Errorf("items length = %v, want 2", len(itms))
	}
}

func TestBadRequest_returns400(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	BadRequest(w, req, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeResponse(t, w)
	if body.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", body.Code, model.ErrBadMessage)
	}
	if body.Msg != "参数错误" {
		t.Errorf("msg = %q, want %q", body.Msg, "参数错误")
	}
}

func TestNotFound_returns404(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	NotFound(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
	body := decodeResponse(t, w)
	if body.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", body.Code, model.ErrNotFound)
	}
	if body.Msg != "资源不存在" {
		t.Errorf("msg = %q, want %q", body.Msg, "资源不存在")
	}
}

func TestUnauthorized_returns401(t *testing.T) {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	Unauthorized(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	body := decodeResponse(t, w)
	if body.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", body.Code, model.ErrNoPermission)
	}
	if body.Msg != "未授权" {
		t.Errorf("msg = %q, want %q", body.Msg, "未授权")
	}
}

// =========================================================================
// Tests: UserHandler
// =========================================================================

// newTestUserHandler creates a UserHandler with fresh mocks.
func newTestUserHandler(authUserRepo *testAuthUserRepo) (*UserHandler, *auth.Service, *mockUserRepo, *mockSessionChecker) {
	authSvc := auth.NewService("test-jwt-secret", 24, 168, authUserRepo, nil, func() int64 { return time.Now().UnixNano() })
	userRepo := &mockUserRepo{}
	sessMgr := &mockSessionChecker{}
	return NewUserHandler(authSvc, userRepo, sessMgr), authSvc, userRepo, sessMgr
}

func TestUserHandler_Register_Success(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, _, _ := newTestUserHandler(authUserRepo)

	body := `{"name":"testuser","account":"testaccount","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["name"] != "testuser" {
		t.Errorf("name = %v, want %q", data["name"], "testuser")
	}
	userID, _ := data["user_id"].(string)
	if userID == "" {
		t.Error("user_id is empty")
	}
	if data["account"] != "testaccount" {
		t.Errorf("account = %v, want %q", data["account"], "testaccount")
	}
	token, _ := data["token"].(string)
	if token == "" {
		t.Error("token is empty")
	}

	// Verify the user was persisted in authUserRepo
	_, err := authUserRepo.GetByID(context.Background(), userID)
	if err != nil {
		t.Errorf("user %q not found in authUserRepo: %v", userID, err)
	}
}

func TestUserHandler_Register_EmptyName(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, _, _ := newTestUserHandler(authUserRepo)

	body := `{"name":"","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Register_EmptyPassword(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, _, _ := newTestUserHandler(authUserRepo)

	body := `{"name":"testuser","password":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_Login_Success(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	// Register a user first
	regBody := `{"name":"loginuser","account":"loginaccount","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Register(w, req)

	regResp := decodeResponse(t, w)
	regData := regResp.Data.(map[string]interface{})
	userID := regData["user_id"].(string)
	account := regData["account"].(string)

	// Stub userRepo.GetByID so Login can fetch the user for the response
	userRepo.getByIDFunc = func(_ context.Context, id string) (*model.User, error) {
		return &model.User{ID: id, Name: "loginuser"}, nil
	}

	// Login with correct credentials
	loginBody := `{"account":"` + account + `","password":"testpass"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", strings.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.Login(w2, req2)

	loginResp := decodeResponse(t, w2)
	if loginResp.Code != 0 {
		t.Fatalf("login failed: code=%d msg=%s", loginResp.Code, loginResp.Msg)
	}
	loginData := loginResp.Data.(map[string]interface{})
	if loginData["user_id"].(string) != userID {
		t.Errorf("user_id = %v, want %v", loginData["user_id"], userID)
	}
	if loginData["account"] != account {
		t.Errorf("account = %v, want %v", loginData["account"], account)
	}
	if loginData["token"].(string) == "" {
		t.Error("token is empty")
	}
	if _, ok := loginData["expires_at"]; !ok {
		t.Error("expires_at is missing")
	}
}

func TestUserHandler_Login_WrongPassword(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, _, _ := newTestUserHandler(authUserRepo)

	// Register a user
	regBody := `{"name":"loginuser","account":"loginaccount","password":"testpass"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(regBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.Register(w, req)

	regResp := decodeResponse(t, w)
	account := regResp.Data.(map[string]interface{})["account"].(string)

	// Login with wrong password
	loginBody := `{"account":"` + account + `","password":"wrongpass"}`
	req2 := httptest.NewRequest(http.MethodPost, "/api/v1/users/login", strings.NewReader(loginBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	handler.Login(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w2.Code, http.StatusUnauthorized)
	}
}

func TestUserHandler_GetMe(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, sessMgr := newTestUserHandler(authUserRepo)

	userRepo.getByIDFunc = func(_ context.Context, id string) (*model.User, error) {
		return &model.User{
			ID:     id,
			Name:   "Current User",
			Avatar: "avatar_url",
			Type:   model.UserHuman,
		}, nil
	}
	sessMgr.isOnlineFunc = func(_ context.Context, userID string) bool {
		return true
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req = setAuthCtx(req, "test_user_1")
	w := httptest.NewRecorder()
	handler.GetMe(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["user_id"] != "test_user_1" {
		t.Errorf("user_id = %v, want %q", data["user_id"], "test_user_1")
	}
	if data["name"] != "Current User" {
		t.Errorf("name = %v, want %q", data["name"], "Current User")
	}
	// WriteUserWithDevices sets status to online when sessMgr.IsOnline returns true
	if data["status"].(float64) != float64(model.UserOnline) {
		t.Errorf("status = %v, want %v", data["status"], model.UserOnline)
	}
}

func TestUserHandler_GetMe_NotFound(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	// No getByIDFunc set — userRepo.GetByID returns error
	userRepo.getByIDFunc = func(_ context.Context, id string) (*model.User, error) {
		return nil, fmt.Errorf("not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	req = setAuthCtx(req, "unknown_user")
	w := httptest.NewRecorder()
	handler.GetMe(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserHandler_GetUser(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, sessMgr := newTestUserHandler(authUserRepo)

	userRepo.getByIDFunc = func(_ context.Context, id string) (*model.User, error) {
		return &model.User{ID: id, Name: "Target User", Type: model.UserHuman}, nil
	}
	sessMgr.isOnlineFunc = func(_ context.Context, userID string) bool {
		return false
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/{user_id}", nil)
	req = setChiURLParam(req, "user_id", "target_user")
	w := httptest.NewRecorder()
	handler.GetUser(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["user_id"] != "target_user" {
		t.Errorf("user_id = %v, want %q", data["user_id"], "target_user")
	}
	if data["status"].(float64) != float64(model.UserOffline) {
		t.Errorf("status = %v, want %v", data["status"], model.UserOffline)
	}
}

func TestUserHandler_GetUser_EmptyID(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, _, _ := newTestUserHandler(authUserRepo)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/", nil)
	// No chi URL param set -> userID remains empty
	w := httptest.NewRecorder()
	handler.GetUser(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_GetUser_NotFound(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	userRepo.getByIDFunc = func(_ context.Context, id string) (*model.User, error) {
		return nil, fmt.Errorf("not found")
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/{user_id}", nil)
	req = setChiURLParam(req, "user_id", "nonexistent")
	w := httptest.NewRecorder()
	handler.GetUser(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestUserHandler_Search(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	userRepo.searchFunc = func(_ context.Context, q string, page, size int) ([]*model.User, int, error) {
		return []*model.User{
			{ID: "user_1", Name: "Alice", Type: model.UserHuman},
			{ID: "user_2", Name: "Bob", Type: model.UserHuman},
		}, 2, nil
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=ali", nil)
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Search(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestUserHandler_Search_DefaultPagination(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	userRepo.searchFunc = func(_ context.Context, q string, page, size int) ([]*model.User, int, error) {
		if page != 1 {
			t.Errorf("page = %d, want 1", page)
		}
		if size != 20 {
			t.Errorf("size = %d, want 20", size)
		}
		return nil, 0, nil
	}

	// No page/size query params -> defaults used
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/search?q=test", nil)
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Search(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestUserHandler_BatchGet(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, sessMgr := newTestUserHandler(authUserRepo)

	userRepo.getByIDsFunc = func(_ context.Context, ids []string) (map[string]*model.User, error) {
		return map[string]*model.User{
			"user_1": {ID: "user_1", Name: "Alice", Type: model.UserHuman},
			"user_2": {ID: "user_2", Name: "Bob", Type: model.UserHuman},
		}, nil
	}
	sessMgr.isOnlineFunc = func(_ context.Context, userID string) bool {
		return true
	}

	body := `{"user_ids":["user_1","user_2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handler.BatchGet(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	result, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	users, ok := result["users"].(map[string]interface{})
	if !ok {
		t.Fatal("users is not a map")
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 users, got %d", len(users))
	}
}

func TestUserHandler_UpdateMe(t *testing.T) {
	authUserRepo := &testAuthUserRepo{}
	handler, _, userRepo, _ := newTestUserHandler(authUserRepo)

	var capturedID, capturedName, capturedAvatar, capturedPrimaryColor, capturedSecondaryColor string
	userRepo.updateFunc = func(_ context.Context, id, name, avatar, primaryColor, secondaryColor string) error {
		capturedID = id
		capturedName = name
		capturedAvatar = avatar
		capturedPrimaryColor = primaryColor
		capturedSecondaryColor = secondaryColor
		return nil
	}

	body := `{"name":"newname","avatar":"new_avatar","primary_color":"#FF0000","secondary_color":"#00FF00"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "user_42")
	w := httptest.NewRecorder()
	handler.UpdateMe(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedID != "user_42" {
		t.Errorf("capturedID = %q, want %q", capturedID, "user_42")
	}
	if capturedName != "newname" {
		t.Errorf("capturedName = %q, want %q", capturedName, "newname")
	}
	if capturedAvatar != "new_avatar" {
		t.Errorf("capturedAvatar = %q, want %q", capturedAvatar, "new_avatar")
	}
	if capturedPrimaryColor != "#FF0000" {
		t.Errorf("capturedPrimaryColor = %q, want %q", capturedPrimaryColor, "#FF0000")
	}
	if capturedSecondaryColor != "#00FF00" {
		t.Errorf("capturedSecondaryColor = %q, want %q", capturedSecondaryColor, "#00FF00")
	}
}

// =========================================================================
// Tests: ConvHandler
// =========================================================================

func TestConvHandler_List(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			getUserConvsFunc: func(_ context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error) {
				return []*db.ConvListItem{
					{ConvID: "conv_1", Name: "Chat A", Type: model.ConvP2P, UnreadCount: 0},
					{ConvID: "conv_2", Name: "Chat B", Type: model.ConvGroup, UnreadCount: 0},
				}, 2, nil
			},
		},
		seqCache: &mockConvSeqCache{
			getUnreadCountFunc: func(_ context.Context, userID, convID string) (int64, error) {
				if convID == "conv_1" {
					return 3, nil
				}
				return 7, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.List(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	// Verify unread count was populated by seqCache
	first := items[0].(map[string]interface{})
	if first["unread_count"].(float64) != 3 {
		t.Errorf("first unread_count = %v, want 3", first["unread_count"])
	}
}

func TestConvHandler_List_DefaultPagination(t *testing.T) {
	var capturedPage, capturedSize int
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			getUserConvsFunc: func(_ context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error) {
				capturedPage = page
				capturedSize = size
				return nil, 0, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations", nil)
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.List(w, req)

	if capturedPage != 1 {
		t.Errorf("page = %d, want 1", capturedPage)
	}
	if capturedSize != 20 {
		t.Errorf("size = %d, want 20", capturedSize)
	}
	_ = w
}

func TestConvHandler_GetDetail(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{
					ConvID: convID, Type: model.ConvGroup, Name: "Test Group",
					OwnerID: "owner_1", Avatar: "avatar", CreatedAt: 1000,
				}, nil
			},
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
			getMembersFunc: func(_ context.Context, convID string) ([]*model.ConvMember, error) {
				return []*model.ConvMember{
					{ConvID: convID, UserID: "owner_1", Role: model.ConvRoleOwner, JoinedAt: 100},
					{ConvID: convID, UserID: "user_1", Role: model.ConvRoleMember, JoinedAt: 200},
				}, nil
			},
		},
		seqCache: &mockConvSeqCache{
			getUnreadCountFunc: func(_ context.Context, userID, convID string) (int64, error) {
				return 5, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetDetail(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "conv_1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "conv_1")
	}
	if data["unread_count"].(float64) != 5 {
		t.Errorf("unread_count = %v, want 5", data["unread_count"])
	}
	members, ok := data["members"].([]interface{})
	if !ok || len(members) != 2 {
		t.Fatalf("expected 2 members, got %d", len(members))
	}
}

func TestConvHandler_GetDetail_NotMember(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: convID}, nil
			},
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return false, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestConvHandler_CreateGroup(t *testing.T) {
	var sysMsgConvID, sysMsgBody string
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			createGroupFunc: func(_ context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
				return &model.Conversation{ConvID: "group_123", Name: name}, nil
			},
		},
		sysMsg: &mockSysMsgSender{
			sendSystemMessageFunc: func(_ context.Context, convID, body string) (*model.Message, error) {
				sysMsgConvID = convID
				sysMsgBody = body
				return nil, nil
			},
		},
		idGen: func() int64 { return 1 },
	}

	body := `{"name":"New Group","member_ids":["user_2","user_3"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "owner_1")
	w := httptest.NewRecorder()
	handler.CreateGroup(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "group_123" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "group_123")
	}
	if data["name"] != "New Group" {
		t.Errorf("name = %v, want %q", data["name"], "New Group")
	}
	// Verify system message was sent
	if sysMsgConvID != "group_123" {
		t.Errorf("sysMsg convID = %q, want %q", sysMsgConvID, "group_123")
	}
	if !strings.Contains(sysMsgBody, "owner_1") {
		t.Errorf("sysMsg body = %q, should contain owner_1", sysMsgBody)
	}
}

func TestConvHandler_CreateGroup_EmptyName(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	body := `{"name":"","member_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/group", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.CreateGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestConvHandler_AddMembers(t *testing.T) {
	var sysMsgCallCount int
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			addMemberFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return nil
			},
		},
		sysMsg: &mockSysMsgSender{
			sendSystemMessageFunc: func(_ context.Context, convID, body string) (*model.Message, error) {
				sysMsgCallCount++
				return nil, nil
			},
		},
	}

	body := `{"user_ids":["user_2","user_3"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.AddMembers(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "conv_1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "conv_1")
	}
	if sysMsgCallCount != 2 {
		t.Errorf("sysMsgCallCount = %d, want 2", sysMsgCallCount)
	}
}

func TestConvHandler_AddMembers_EmptyUserIDs(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{},
	}

	body := `{"user_ids":[]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.AddMembers(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestConvHandler_AddMembers_DuplicateUserIDs(t *testing.T) {
	var addMemberCalls []string
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			addMemberFunc: func(_ context.Context, convID, userID, operatorID string) error {
				addMemberCalls = append(addMemberCalls, userID)
				return nil
			},
		},
		sysMsg: &mockSysMsgSender{
			sendSystemMessageFunc: func(_ context.Context, convID, body string) (*model.Message, error) {
				return nil, nil
			},
		},
	}

	// user_2 appears twice, should only trigger AddMember once
	body := `{"user_ids":["user_2","user_3","user_2"]}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/members", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.AddMembers(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if len(addMemberCalls) != 2 {
		t.Errorf("expected 2 AddMember calls (deduped), got %d", len(addMemberCalls))
	}
}

func TestConvHandler_RemoveMember(t *testing.T) {
	var sysMsgConvID, sysMsgBody string
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			removeMemberFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return nil
			},
		},
		sysMsg: &mockSysMsgSender{
			sendSystemMessageFunc: func(_ context.Context, convID, body string) (*model.Message, error) {
				sysMsgConvID = convID
				sysMsgBody = body
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/conversations/{conv_id}/members/{user_id}", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setChiURLParam(req, "user_id", "user_2")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.RemoveMember(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if sysMsgConvID != "conv_1" {
		t.Errorf("sysMsg convID = %q, want %q", sysMsgConvID, "conv_1")
	}
	if !strings.Contains(sysMsgBody, "user_2") {
		t.Errorf("sysMsg body = %q, should contain user_2", sysMsgBody)
	}
}

func TestConvHandler_RemoveMember_Error(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			removeMemberFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return &model.AppError{Code: model.ErrNoPermission, Message: "no permission"}
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/conversations/{conv_id}/members/{user_id}", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setChiURLParam(req, "user_id", "user_2")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.RemoveMember(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestConvHandler_Leave(t *testing.T) {
	var sysMsgConvID string
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			leaveFunc: func(_ context.Context, convID, userID string) error {
				return nil
			},
		},
		sysMsg: &mockSysMsgSender{
			sendSystemMessageFunc: func(_ context.Context, convID, body string) (*model.Message, error) {
				sysMsgConvID = convID
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/leave", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.Leave(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if sysMsgConvID != "conv_1" {
		t.Errorf("sysMsg convID = %q, want %q", sysMsgConvID, "conv_1")
	}
}

func TestConvHandler_MarkRead(t *testing.T) {
	var capturedUserID, capturedConvID string
	var capturedMsgID int64
	handler := &ConvHandler{
		readMarker: &mockReadMarker{
			markReadFunc: func(_ context.Context, userID, convID string, msgID int64) error {
				capturedUserID = userID
				capturedConvID = convID
				capturedMsgID = msgID
				return nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	body := `{"msg_id":100}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/read", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.MarkRead(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedUserID != "user_1" {
		t.Errorf("capturedUserID = %q, want %q", capturedUserID, "user_1")
	}
	if capturedConvID != "conv_1" {
		t.Errorf("capturedConvID = %q, want %q", capturedConvID, "conv_1")
	}
	if capturedMsgID != 100 {
		t.Errorf("capturedMsgID = %d, want 100", capturedMsgID)
	}
}

func TestConvHandler_MarkRead_NilMarker(t *testing.T) {
	// readMarker is nil — handler should not panic
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	body := `{"msg_id":50}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/read", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.MarkRead(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestConvHandler_UpdateGroup(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{
					ConvID: convID,
					Type:   model.ConvGroup,
				}, nil
			},
			getMemberRoleFunc: func(_ context.Context, convID, userID string) (model.ConvRole, error) {
				return model.ConvRoleOwner, nil
			},
		},
		convRepo: &mockConvDataRepo{
			updateNameAvatarFunc: func(_ context.Context, convID, name, avatar string) error {
				return nil
			},
		},
	}

	body := `{"name":"Updated Group","avatar":"new_avatar"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/conversations/{conv_id}", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.UpdateGroup(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["name"] != "Updated Group" {
		t.Errorf("name = %v, want %q", data["name"], "Updated Group")
	}
	if data["avatar"] != "new_avatar" {
		t.Errorf("avatar = %v, want %q", data["avatar"], "new_avatar")
	}
}

func TestConvHandler_UpdateGroup_NotGroup(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{
					ConvID: convID,
					Type:   model.ConvP2P, // not a group
				}, nil
			},
		},
	}

	body := `{"name":"New Name"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/conversations/{conv_id}", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.UpdateGroup(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// Tests: MsgHandler
// =========================================================================

func TestMsgHandler_GetHistory(t *testing.T) {
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getHistoryFunc: func(_ context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
				return []*model.Message{
					{MsgID: 3, ConvID: convID, SenderID: "user_1", Body: "third", ContentType: model.ContentText, ConvSeq: 3, Status: model.MsgSent},
					{MsgID: 2, ConvID: convID, SenderID: "user_1", Body: "second"},
					{MsgID: 1, ConvID: convID, SenderID: "user_2", Body: "first"},
				}, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/messages", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetHistory(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	items, ok := resp.Data.([]interface{})
	if !ok || len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	first := items[0].(map[string]interface{})
	if first["msg_id"].(float64) != 3 {
		t.Errorf("first msg_id = %v, want 3", first["msg_id"])
	}
	if first["body"] != "third" {
		t.Errorf("first body = %v, want %q", first["body"], "third")
	}
}

func TestMsgHandler_GetHistory_WithBefore(t *testing.T) {
	var capturedBefore int64
	var capturedLimit int
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getHistoryFunc: func(_ context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
				capturedBefore = beforeMsgID
				capturedLimit = limit
				return nil, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/messages?before_msg_id=100&limit=10", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetHistory(w, req)

	if capturedBefore != 100 {
		t.Errorf("capturedBefore = %d, want 100", capturedBefore)
	}
	if capturedLimit != 10 {
		t.Errorf("capturedLimit = %d, want 10", capturedLimit)
	}
	_ = w
}

func TestMsgHandler_GetHistory_DefaultLimit(t *testing.T) {
	var capturedLimit int
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getHistoryFunc: func(_ context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
				capturedLimit = limit
				return nil, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/messages", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetHistory(w, req)

	if capturedLimit != 50 {
		t.Errorf("capturedLimit = %d, want 50", capturedLimit)
	}
	_ = w
}

func TestMsgHandler_GetHistory_NotMember(t *testing.T) {
	handler := &MsgHandler{
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return false, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/messages", nil)
	req = setChiURLParam(req, "conv_id", "conv_1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetHistory(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

// =========================================================================
// Tests: ContactHandler
// =========================================================================

func TestContactHandler_List(t *testing.T) {
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			listFunc: func(_ context.Context, userID string, page, size int) ([]*model.Contact, int, error) {
				return []*model.Contact{
					{UserID: userID, ContactID: "contact_1", Nickname: "Friend1", AddedAt: 1000},
					{UserID: userID, ContactID: "contact_2", Nickname: "Friend2", AddedAt: 2000},
				}, 2, nil
			},
		},
		userRepo: &mockUserQueryRepo{
			getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
				return map[string]*model.User{
					"contact_1": {ID: "contact_1", Name: "Friend One", Avatar: "avatar1"},
					"contact_2": {ID: "contact_2", Name: "Friend Two", Avatar: "avatar2"},
				}, nil
			},
		},
		sessMgr: &mockSessionChecker{
			isOnlineFunc: func(_ context.Context, userID string) bool {
				return userID == "contact_1"
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contacts", nil)
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.List(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Data is not a map")
	}
	if data["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", data["total"])
	}
	items, ok := data["items"].([]interface{})
	if !ok || len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0].(map[string]interface{})
	if first["user_id"] != "contact_1" {
		t.Errorf("first user_id = %v, want %q", first["user_id"], "contact_1")
	}
	// contact_1 should be online
	if first["status"].(float64) != float64(model.UserOnline) {
		t.Errorf("first status = %v, want %v", first["status"], model.UserOnline)
	}
	// contact_1 should have enriched name and avatar
	if first["name"] != "Friend One" {
		t.Errorf("first name = %v, want %q", first["name"], "Friend One")
	}
}

func TestContactHandler_Add(t *testing.T) {
	var capturedContact *model.Contact
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			addFunc: func(_ context.Context, c *model.Contact) error {
				capturedContact = c
				return nil
			},
		},
	}

	body := `{"user_id":"contact_1","nickname":"Buddy"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Add(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedContact == nil {
		t.Fatal("capturedContact is nil")
	}
	if capturedContact.UserID != "test_user" {
		t.Errorf("UserID = %q, want %q", capturedContact.UserID, "test_user")
	}
	if capturedContact.ContactID != "contact_1" {
		t.Errorf("ContactID = %q, want %q", capturedContact.ContactID, "contact_1")
	}
	if capturedContact.Nickname != "Buddy" {
		t.Errorf("Nickname = %q, want %q", capturedContact.Nickname, "Buddy")
	}
	if capturedContact.AddedAt == 0 {
		t.Error("AddedAt is 0 — expected a timestamp")
	}
}

func TestContactHandler_Add_EmptyUserID(t *testing.T) {
	handler := &ContactHandler{}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Add(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Add_Self(t *testing.T) {
	handler := &ContactHandler{}

	body := `{"user_id":"test_user"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contacts", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Add(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_Remove(t *testing.T) {
	var capturedUserID, capturedContactID string
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			removeFunc: func(_ context.Context, userID, contactID string) error {
				capturedUserID = userID
				capturedContactID = contactID
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/{user_id}", nil)
	req = setChiURLParam(req, "user_id", "contact_1")
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.Remove(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedUserID != "test_user" {
		t.Errorf("capturedUserID = %q, want %q", capturedUserID, "test_user")
	}
	if capturedContactID != "contact_1" {
		t.Errorf("capturedContactID = %q, want %q", capturedContactID, "contact_1")
	}
}

func TestContactHandler_UpdateNickname(t *testing.T) {
	var capturedUserID, capturedContactID, capturedNickname string
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			updateNicknameFunc: func(_ context.Context, userID, contactID, nickname string) error {
				capturedUserID = userID
				capturedContactID = contactID
				capturedNickname = nickname
				return nil
			},
		},
	}

	body := `{"nickname":"New Nickname"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/contacts/{user_id}", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "user_id", "contact_1")
	req = setAuthCtx(req, "test_user")
	w := httptest.NewRecorder()
	handler.UpdateNickname(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedUserID != "test_user" {
		t.Errorf("capturedUserID = %q, want %q", capturedUserID, "test_user")
	}
	if capturedContactID != "contact_1" {
		t.Errorf("capturedContactID = %q, want %q", capturedContactID, "contact_1")
	}
	if capturedNickname != "New Nickname" {
		t.Errorf("capturedNickname = %q, want %q", capturedNickname, "New Nickname")
	}
}

// ---------------------------------------------------------------------------
// Join request handler tests
// ---------------------------------------------------------------------------

func TestConvHandler_RequestJoin(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			requestJoinFunc: func(_ context.Context, convID, userID string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.RequestJoin(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "g1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "g1")
	}
}

func TestConvHandler_RequestJoin_Error(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			requestJoinFunc: func(_ context.Context, convID, userID string) error {
				return &model.AppError{Code: model.ErrBadMessage, Message: "join error", Key: "err.already_member"}
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.RequestJoin(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrBadMessage)
	}
}

func TestConvHandler_ListJoinRequests(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			listJoinRequestsFunc: func(_ context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
				return []*model.JoinRequest{
					{ConvID: "g1", UserID: "bob", Status: model.JoinRequestPending},
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "admin")
	w := httptest.NewRecorder()
	handler.ListJoinRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestConvHandler_ListJoinRequests_Empty(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			listJoinRequestsFunc: func(_ context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
				return []*model.JoinRequest{}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "admin")
	w := httptest.NewRecorder()
	handler.ListJoinRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	list, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected array in data")
	}
	if len(list) != 0 {
		t.Errorf("expected empty list, got %d items", len(list))
	}
}

func TestConvHandler_ApproveJoinRequest(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			approveJoinRequestFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/join-requests/{user_id}/approve", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setChiURLParam(req, "user_id", "bob")
	req = setAuthCtx(req, "admin")
	w := httptest.NewRecorder()
	handler.ApproveJoinRequest(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "g1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "g1")
	}
	if data["user_id"] != "bob" {
		t.Errorf("user_id = %v, want %q", data["user_id"], "bob")
	}
}

func TestConvHandler_RejectJoinRequest(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			rejectJoinRequestFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/join-requests/{user_id}/reject", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setChiURLParam(req, "user_id", "bob")
	req = setAuthCtx(req, "admin")
	w := httptest.NewRecorder()
	handler.RejectJoinRequest(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "g1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "g1")
	}
	if data["user_id"] != "bob" {
		t.Errorf("user_id = %v, want %q", data["user_id"], "bob")
	}
}
