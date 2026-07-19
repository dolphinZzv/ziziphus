package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"ziziphus/internal/auth"
	"ziziphus/internal/gateway"
	"ziziphus/internal/storage/db"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/model"
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

func (r *testAuthUserRepo) GetByEmail(_ context.Context, email string) (*model.User, error) {
	if r.users == nil {
		return nil, fmt.Errorf("user not found")
	}
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *testAuthUserRepo) GetByGithubID(_ context.Context, githubID string) (*model.User, error) {
	for _, u := range r.users {
		if u.GithubID == githubID {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *testAuthUserRepo) GetByGoogleID(_ context.Context, googleID string) (*model.User, error) {
	for _, u := range r.users {
		if u.GoogleID == googleID {
			return u, nil
		}
	}
	return nil, fmt.Errorf("user not found")
}

func (r *testAuthUserRepo) UpdateOAuthID(_ context.Context, userID, provider, oauthID string) error {
	u, ok := r.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	switch provider {
	case "github":
		u.GithubID = oauthID
	case "google":
		u.GoogleID = oauthID
	}
	return nil
}

func (r *testAuthUserRepo) ClearOAuthID(_ context.Context, userID, provider string) error {
	u, ok := r.users[userID]
	if !ok {
		return fmt.Errorf("user not found")
	}
	switch provider {
	case "github":
		u.GithubID = ""
	case "google":
		u.GoogleID = ""
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: userRepo (for UserHandler)
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	createFunc            func(ctx context.Context, u *model.User) error
	getByIDFunc           func(ctx context.Context, id string) (*model.User, error)
	getByIDsFunc          func(ctx context.Context, ids []string) (map[string]*model.User, error)
	searchFunc            func(ctx context.Context, q string, page, size int) ([]*model.User, int, error)
	updateFunc            func(ctx context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor string, discoverable, allowDirectChat bool) error
	countAgentsFunc       func(ctx context.Context, uid string) (int, error)
	listAgentsFunc        func(ctx context.Context, uid string) ([]*model.User, error)
	updateAgentFunc       func(ctx context.Context, agentID, uid, name, avatar, cover, primaryColor, secondaryColor string, wakeMode model.WakeMode, discoverable, allowDirectChat bool) error
	deleteAgentFunc       func(ctx context.Context, agentID, uid string) error
	getByAPIKeyFunc       func(ctx context.Context, apiKey string) (*model.User, error)
	updateAgentAPIKeyFunc func(ctx context.Context, agentID, uid, apiKey string) error
	deleteAccountFunc     func(ctx context.Context, userID string) error
	updateLanguageFunc    func(ctx context.Context, userID, language string) error
	getByEmailFunc        func(ctx context.Context, email string) (*model.User, error)
	updatePasswordFunc    func(ctx context.Context, userID, password string) error
	getByAccountFunc      func(ctx context.Context, account string) (*model.User, error)
	getByGithubIDFunc     func(ctx context.Context, githubID string) (*model.User, error)
	getByGoogleIDFunc     func(ctx context.Context, googleID string) (*model.User, error)
	updateOAuthIDFunc     func(ctx context.Context, userID, provider, oauthID string) error
	clearOAuthIDFunc      func(ctx context.Context, userID, provider string) error
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

func (m *mockUserRepo) Update(ctx context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor, headline string, discoverable, allowDirectChat bool) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, id, name, avatar, cover, email, primaryColor, secondaryColor, discoverable, allowDirectChat)
	}
	return nil
}

func (m *mockUserRepo) CountAgents(ctx context.Context, uid string) (int, error) {
	if m.countAgentsFunc != nil {
		return m.countAgentsFunc(ctx, uid)
	}
	return 0, nil
}

func (m *mockUserRepo) ListAgents(ctx context.Context, uid string) ([]*model.User, error) {
	if m.listAgentsFunc != nil {
		return m.listAgentsFunc(ctx, uid)
	}
	return nil, nil
}

func (m *mockUserRepo) UpdateAgent(ctx context.Context, agentID, uid, name, avatar, cover, primaryColor, secondaryColor, headline string, wakeMode model.WakeMode, discoverable, allowDirectChat bool) error {
	if m.updateAgentFunc != nil {
		return m.updateAgentFunc(ctx, agentID, uid, name, avatar, cover, primaryColor, secondaryColor, wakeMode, discoverable, allowDirectChat)
	}
	return nil
}

func (m *mockUserRepo) DeleteAgent(ctx context.Context, agentID, uid string) error {
	if m.deleteAgentFunc != nil {
		return m.deleteAgentFunc(ctx, agentID, uid)
	}
	return nil
}

func (m *mockUserRepo) GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error) {
	if m.getByAPIKeyFunc != nil {
		return m.getByAPIKeyFunc(ctx, apiKey)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockUserRepo) UpdateAgentAPIKey(ctx context.Context, agentID, uid, apiKey string) error {
	if m.updateAgentAPIKeyFunc != nil {
		return m.updateAgentAPIKeyFunc(ctx, agentID, uid, apiKey)
	}
	return nil
}

func (m *mockUserRepo) DeleteAccount(ctx context.Context, userID string) error {
	if m.deleteAccountFunc != nil {
		return m.deleteAccountFunc(ctx, userID)
	}
	return nil
}

func (m *mockUserRepo) UpdateLanguage(ctx context.Context, userID, language string) error {
	if m.updateLanguageFunc != nil {
		return m.updateLanguageFunc(ctx, userID, language)
	}
	return nil
}

func (m *mockUserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	if m.getByEmailFunc != nil {
		return m.getByEmailFunc(ctx, email)
	}
	return nil, fmt.Errorf("not found")
}

func (m *mockUserRepo) UpdatePassword(ctx context.Context, userID, password string) error {
	if m.updatePasswordFunc != nil {
		return m.updatePasswordFunc(ctx, userID, password)
	}
	return nil
}

func (m *mockUserRepo) GetByGithubID(ctx context.Context, githubID string) (*model.User, error) {
	if m.getByGithubIDFunc != nil {
		return m.getByGithubIDFunc(ctx, githubID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepo) GetByGoogleID(ctx context.Context, googleID string) (*model.User, error) {
	if m.getByGoogleIDFunc != nil {
		return m.getByGoogleIDFunc(ctx, googleID)
	}
	return nil, fmt.Errorf("not implemented")
}

func (m *mockUserRepo) UpdateOAuthID(ctx context.Context, userID, provider, oauthID string) error {
	if m.updateOAuthIDFunc != nil {
		return m.updateOAuthIDFunc(ctx, userID, provider, oauthID)
	}
	return nil
}

func (m *mockUserRepo) ClearOAuthID(ctx context.Context, userID, provider string) error {
	if m.clearOAuthIDFunc != nil {
		return m.clearOAuthIDFunc(ctx, userID, provider)
	}
	return nil
}

func (m *mockUserRepo) GetByAccount(ctx context.Context, account string) (*model.User, error) {
	if m.getByAccountFunc != nil {
		return m.getByAccountFunc(ctx, account)
	}
	return nil, fmt.Errorf("not found")
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
	createGroupFunc        func(ctx context.Context, name, headline, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error)
	getOrCreateP2PFunc     func(ctx context.Context, userA, userB string) (*model.Conversation, error)
	addMemberFunc          func(ctx context.Context, convID, userID, operatorID string) error
	removeMemberFunc       func(ctx context.Context, convID, userID, operatorID string) error
	leaveFunc              func(ctx context.Context, convID, userID string) error
	getMembersFunc         func(ctx context.Context, convID string) ([]*model.ConvMember, error)
	isMemberFunc           func(ctx context.Context, convID, userID string) (bool, error)
	requestJoinFunc        func(ctx context.Context, convID, userID string) (bool, error)
	listJoinRequestsFunc   func(ctx context.Context, convID, operatorID string) ([]*model.JoinRequest, error)
	approveJoinRequestFunc func(ctx context.Context, convID, userID, operatorID string) error
	rejectJoinRequestFunc  func(ctx context.Context, convID, userID, operatorID string) error
	getMemberRoleFunc      func(ctx context.Context, convID, userID string) (model.ConvRole, error)
	disbandFunc            func(ctx context.Context, convID, ownerID string) error
}

func (m *mockConvManager) Get(ctx context.Context, convID string) (*model.Conversation, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, convID)
	}
	return nil, &model.AppError{Code: model.ErrNotFound, Message: "not found"}
}

func (m *mockConvManager) CreateGroup(ctx context.Context, name, headline, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
	if m.createGroupFunc != nil {
		return m.createGroupFunc(ctx, name, headline, ownerID, memberIDs, idGen)
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

func (m *mockConvManager) RequestJoin(ctx context.Context, convID, userID string) (bool, error) {
	if m.requestJoinFunc != nil {
		return m.requestJoinFunc(ctx, convID, userID)
	}
	return false, nil
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

func (m *mockConvManager) Disband(ctx context.Context, convID, ownerID string) error {
	if m.disbandFunc != nil {
		return m.disbandFunc(ctx, convID, ownerID)
	}
	return nil
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
	getUserConvsFunc       func(ctx context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error)
	updateNameAvatarFunc   func(ctx context.Context, convID, name, avatar string) error
	updateNoticeFunc       func(ctx context.Context, convID, notice string) error
	updateCoverFunc        func(ctx context.Context, convID, cover string) error
	updatePrimaryColorFunc func(ctx context.Context, convID, color string) error
	searchByNameFunc       func(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error)
	pinFunc                func(ctx context.Context, userID, convID string) error
	unpinFunc              func(ctx context.Context, userID, convID string) error
	cloneFunc              func(ctx context.Context, src, dst, owner, name string, idGen func() int64) error
	areContactsFunc        func(ctx context.Context, userA, userB string) (bool, error)
	getSettingsFunc        func(ctx context.Context, convID string) (map[string]any, error)
	updateSettingsFunc     func(ctx context.Context, convID string, settings map[string]any) error
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
	if m.updateNoticeFunc != nil {
		return m.updateNoticeFunc(ctx, convID, notice)
	}
	return nil
}

func (m *mockConvDataRepo) UpdateCover(ctx context.Context, convID, cover string) error {
	if m.updateCoverFunc != nil {
		return m.updateCoverFunc(ctx, convID, cover)
	}
	return nil
}

func (m *mockConvDataRepo) UpdatePrimaryColor(ctx context.Context, convID, color string) error {
	if m.updatePrimaryColorFunc != nil {
		return m.updatePrimaryColorFunc(ctx, convID, color)
	}
	return nil
}

func (m *mockConvDataRepo) Pin(ctx context.Context, userID, convID string) error {
	if m.pinFunc != nil {
		return m.pinFunc(ctx, userID, convID)
	}
	return nil
}
func (m *mockConvDataRepo) Unpin(ctx context.Context, userID, convID string) error {
	if m.unpinFunc != nil {
		return m.unpinFunc(ctx, userID, convID)
	}
	return nil
}
func (m *mockConvDataRepo) Clone(ctx context.Context, src, dst, owner, name string, idGen func() int64) error {
	if m.cloneFunc != nil {
		return m.cloneFunc(ctx, src, dst, owner, name, idGen)
	}
	return nil
}

func (m *mockConvDataRepo) AreContacts(ctx context.Context, userA, userB string) (bool, error) {
	if m.areContactsFunc != nil {
		return m.areContactsFunc(ctx, userA, userB)
	}
	return false, nil
}

func (m *mockConvDataRepo) GetSettings(ctx context.Context, convID string) (map[string]any, error) {
	if m.getSettingsFunc != nil {
		return m.getSettingsFunc(ctx, convID)
	}
	return nil, nil
}

func (m *mockConvDataRepo) UpdateHeadline(ctx context.Context, convID, headline string) error {
	return nil
}

func (m *mockConvDataRepo) UpdateSettings(ctx context.Context, convID string, settings map[string]any) error {
	if m.updateSettingsFunc != nil {
		return m.updateSettingsFunc(ctx, convID, settings)
	}
	return nil
}

func (m *mockConvDataRepo) SearchByName(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error) {
	if m.searchByNameFunc != nil {
		return m.searchByNameFunc(ctx, q, page, size)
	}
	return nil, 0, nil
}

func (m *mockConvDataRepo) GetByShareToken(ctx context.Context, shareToken string) (*model.Conversation, error) {
	return nil, fmt.Errorf("not found")
}

func (m *mockConvDataRepo) GetMemberCount(ctx context.Context, convID string) (int, error) {
	return 0, nil
}

func (m *mockConvDataRepo) GenerateShareToken(ctx context.Context, convID string) (string, error) {
	return "test-token", nil
}

func (m *mockConvDataRepo) RemoveShareToken(ctx context.Context, convID string) error {
	return nil
}

// ---------------------------------------------------------------------------
// Mock: msgStorage
// ---------------------------------------------------------------------------

type mockMsgStorage struct {
	getHistoryFunc func(ctx context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error)
	getFunc        func(ctx context.Context, msgID int64) (*model.Message, error)
}

func (m *mockMsgStorage) GetHistory(ctx context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
	if m.getHistoryFunc != nil {
		return m.getHistoryFunc(ctx, convID, beforeMsgID, aroundMsgID, limit, keyword, startDate, endDate)
	}
	return nil, nil
}

func (m *mockMsgStorage) Get(ctx context.Context, msgID int64) (*model.Message, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, msgID)
	}
	return nil, nil
}

// ---------------------------------------------------------------------------
// Mock: receiptStorage
// ---------------------------------------------------------------------------

type mockReceiptStorage struct {
	getByMsgIDFunc func(ctx context.Context, msgID int64) ([]*model.Receipt, error)
}

func (m *mockReceiptStorage) GetByMsgID(ctx context.Context, msgID int64) ([]*model.Receipt, error) {
	if m.getByMsgIDFunc != nil {
		return m.getByMsgIDFunc(ctx, msgID)
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

// ---------------------------------------------------------------------------
// Mock: userGetter (for ConvHandler)
// ---------------------------------------------------------------------------

type mockUserGetter struct {
	getByIDFunc func(ctx context.Context, id string) (*model.User, error)
}

func (m *mockUserGetter) GetByID(ctx context.Context, id string) (*model.User, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return &model.User{ID: id, Name: id, Type: model.UserHuman}, nil
}

// ---------------------------------------------------------------------------
// Mock: contactRequestStorage (for ContactHandler)
// ---------------------------------------------------------------------------

type mockContactRequestStorage struct {
	insertFunc             func(ctx context.Context, req *model.ContactRequest) (int64, error)
	updateFormMsgIDFunc    func(ctx context.Context, id, formMsgID int64) error
	getByIDFunc            func(ctx context.Context, id int64) (*model.ContactRequest, error)
	getByFormMsgIDFunc     func(ctx context.Context, formMsgID int64) (*model.ContactRequest, error)
	getByPairFunc          func(ctx context.Context, fromUserID, toUserID string) (*model.ContactRequest, error)
	listSentFunc           func(ctx context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error)
	listReceivedFunc       func(ctx context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error)
	deleteFunc             func(ctx context.Context, id int64) error
	existsAnyDirectionFunc func(ctx context.Context, userA, userB string) (bool, error)
}

func (m *mockContactRequestStorage) Insert(ctx context.Context, req *model.ContactRequest) (int64, error) {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, req)
	}
	return 1, nil
}
func (m *mockContactRequestStorage) UpdateFormMsgID(ctx context.Context, id, formMsgID int64) error {
	if m.updateFormMsgIDFunc != nil {
		return m.updateFormMsgIDFunc(ctx, id, formMsgID)
	}
	return nil
}
func (m *mockContactRequestStorage) GetByID(ctx context.Context, id int64) (*model.ContactRequest, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}
func (m *mockContactRequestStorage) GetByFormMsgID(ctx context.Context, formMsgID int64) (*model.ContactRequest, error) {
	if m.getByFormMsgIDFunc != nil {
		return m.getByFormMsgIDFunc(ctx, formMsgID)
	}
	return nil, nil
}
func (m *mockContactRequestStorage) GetByPair(ctx context.Context, fromUserID, toUserID string) (*model.ContactRequest, error) {
	if m.getByPairFunc != nil {
		return m.getByPairFunc(ctx, fromUserID, toUserID)
	}
	return nil, nil
}
func (m *mockContactRequestStorage) ListSent(ctx context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error) {
	if m.listSentFunc != nil {
		return m.listSentFunc(ctx, userID, page, size)
	}
	return nil, 0, nil
}
func (m *mockContactRequestStorage) ListReceived(ctx context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error) {
	if m.listReceivedFunc != nil {
		return m.listReceivedFunc(ctx, userID, status, page, size)
	}
	return nil, 0, nil
}
func (m *mockContactRequestStorage) Delete(ctx context.Context, id int64) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}
func (m *mockContactRequestStorage) ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error) {
	if m.existsAnyDirectionFunc != nil {
		return m.existsAnyDirectionFunc(ctx, userA, userB)
	}
	return false, nil
}

type mockFormMessageSender struct {
	sendFormMessageFunc   func(ctx context.Context, convID string, body *model.FormDefinitionBody) (*model.Message, error)
	sendSystemMessageFunc func(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error)
}

func (m *mockFormMessageSender) SendFormMessage(ctx context.Context, convID string, body *model.FormDefinitionBody) (*model.Message, error) {
	if m.sendFormMessageFunc != nil {
		return m.sendFormMessageFunc(ctx, convID, body)
	}
	return &model.Message{MsgID: 1}, nil
}
func (m *mockFormMessageSender) SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error) {
	if m.sendSystemMessageFunc != nil {
		return m.sendSystemMessageFunc(ctx, convID, body, senderID...)
	}
	return &model.Message{MsgID: 2}, nil
}

type mockSystemConvManager struct {
	getOrCreateSystemConvFunc func(ctx context.Context, userID string) (*model.Conversation, error)
	leaveFunc                 func(ctx context.Context, convID, userID string) error
}

func (m *mockSystemConvManager) GetOrCreateSystemConv(ctx context.Context, userID string) (*model.Conversation, error) {
	if m.getOrCreateSystemConvFunc != nil {
		return m.getOrCreateSystemConvFunc(ctx, userID)
	}
	return &model.Conversation{ConvID: "sys:test"}, nil
}

func (m *mockSystemConvManager) Leave(ctx context.Context, convID, userID string) error {
	if m.leaveFunc != nil {
		return m.leaveFunc(ctx, convID, userID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: sessionManager (for SessionHandler)
// ---------------------------------------------------------------------------

type mockSessionManager struct {
	getUserSessionIDsFunc func(ctx context.Context, userID string) []string
	getFunc               func(ctx context.Context, sessionID string) *model.Session
	deleteFunc            func(ctx context.Context, sessionID string) error
}

func (m *mockSessionManager) GetUserSessionIDs(ctx context.Context, userID string) []string {
	if m.getUserSessionIDsFunc != nil {
		return m.getUserSessionIDsFunc(ctx, userID)
	}
	return nil
}

func (m *mockSessionManager) Get(ctx context.Context, sessionID string) *model.Session {
	if m.getFunc != nil {
		return m.getFunc(ctx, sessionID)
	}
	return nil
}

func (m *mockSessionManager) Delete(ctx context.Context, sessionID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, sessionID)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Mock: fileDB
// ---------------------------------------------------------------------------

type mockFileDB struct {
	insertFunc            func(ctx context.Context, f *model.FileInfo) error
	getByIDFunc           func(ctx context.Context, fileID string) (*model.FileInfo, error)
	listByConvIDFunc      func(ctx context.Context, convID string, page, size int) ([]*model.FileInfo, int, error)
	deleteByIDFunc        func(ctx context.Context, fileID, uploaderID string) error
	listFilesInFolderFunc func(ctx context.Context, convID, folderPath string, page, size int) ([]*model.FileInfo, int, error)
	updateFolderPathFunc  func(ctx context.Context, fileID, folderPath string) error
}

func (m *mockFileDB) Insert(ctx context.Context, f *model.FileInfo) error {
	if m.insertFunc != nil {
		return m.insertFunc(ctx, f)
	}
	return nil
}
func (m *mockFileDB) GetByID(ctx context.Context, fileID string) (*model.FileInfo, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, fileID)
	}
	return &model.FileInfo{FileID: fileID, Name: fileID + ".png", URL: "/files/" + fileID + ".png", Size: 100, ContentType: 1}, nil
}
func (m *mockFileDB) ListByConvID(ctx context.Context, convID string, page, size int) ([]*model.FileInfo, int, error) {
	if m.listByConvIDFunc != nil {
		return m.listByConvIDFunc(ctx, convID, page, size)
	}
	return nil, 0, nil
}
func (m *mockFileDB) DeleteByID(ctx context.Context, fileID, uploaderID string) error {
	if m.deleteByIDFunc != nil {
		return m.deleteByIDFunc(ctx, fileID, uploaderID)
	}
	return nil
}
func (m *mockFileDB) ListFilesInFolder(ctx context.Context, convID, folderPath string, page, size int) ([]*model.FileInfo, int, error) {
	if m.listFilesInFolderFunc != nil {
		return m.listFilesInFolderFunc(ctx, convID, folderPath, page, size)
	}
	return nil, 0, nil
}
func (m *mockFileDB) UpdateFolderPath(ctx context.Context, fileID, folderPath string) error {
	if m.updateFolderPathFunc != nil {
		return m.updateFolderPathFunc(ctx, fileID, folderPath)
	}
	return nil
}

// ---------------------------------------------------------------------------
// decodeResponse decodes an *httptest.ResponseRecorder body into APIResponse.
// ---------------------------------------------------------------------------

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) APIResponse {
	t.Helper()
	var resp APIResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// ---------------------------------------------------------------------------
// Mock: mfaStorage
// ---------------------------------------------------------------------------

type mockMFAStorage struct{}

func (m *mockMFAStorage) Get(_ context.Context, _ string) (*model.UserMFA, error) { return nil, nil }
func (m *mockMFAStorage) Upsert(_ context.Context, _ *model.UserMFA) error        { return nil }
func (m *mockMFAStorage) Disable(_ context.Context, _ string) error               { return nil }

// ---------------------------------------------------------------------------
// Mock: emailVerifyHandler
// ---------------------------------------------------------------------------

type mockEmailVerifyHandler struct{}

func (m *mockEmailVerifyHandler) Upsert(_ context.Context, _ *model.EmailVerify) error { return nil }
func (m *mockEmailVerifyHandler) Get(_ context.Context, _ string) (*model.EmailVerify, error) {
	return nil, nil
}
func (m *mockEmailVerifyHandler) Delete(_ context.Context, _ string) error { return nil }

// ---------------------------------------------------------------------------
// Mock: emailSender
// ---------------------------------------------------------------------------

type mockEmailSender struct{}

func (m *mockEmailSender) Enabled() bool                           { return false }
func (m *mockEmailSender) SendVerificationCode(_, _ string) error  { return nil }
func (m *mockEmailSender) SendPasswordResetCode(_, _ string) error { return nil }

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func setAuthCtx(r *http.Request, userID string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), auth.CtxKeyUserID, userID))
}

func setChiURLParam(r *http.Request, key, value string) *http.Request {
	chiCtx := chi.RouteContext(r.Context())
	if chiCtx == nil {
		chiCtx = chi.NewRouteContext()
		r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, chiCtx))
	}
	chiCtx.URLParams.Add(key, value)
	return r
}

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

func TestHandlers_DetectLanguage(t *testing.T) {
	tests := []struct {
		name   string
		accept string
		want   string
	}{
		{name: "empty header returns zh", accept: "", want: "zh"},
		{name: "accepts en", accept: "en-US,en;q=0.9", want: "en"},
		{name: "accepts zh", accept: "zh-CN,zh;q=0.8", want: "zh"},
		{name: "accepts ja", accept: "ja-JP", want: "ja"},
		{name: "accepts fr", accept: "fr-FR", want: "fr"},
		{name: "multiple with en first", accept: "en;q=0.1,zh;q=0.9", want: "en"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Handlers{}
			handler := i18n.Middleware(http.HandlerFunc(h.DetectLanguage))
			req := httptest.NewRequest(http.MethodGet, "/api/v1/i18n/detect", nil)
			if tt.accept != "" {
				req.Header.Set("Accept-Language", tt.accept)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, req)

			resp := decodeResponse(t, w)
			if resp.Code != 0 {
				t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
			}
			data, ok := resp.Data.(map[string]interface{})
			if !ok {
				t.Fatal("Data is not a map")
			}
			got, _ := data["language"].(string)
			if got != tt.want {
				t.Errorf("language = %q, want %q", got, tt.want)
			}
		})
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
	return NewUserHandler(authSvc, userRepo, sessMgr, func() int64 { return time.Now().UnixNano() }, &mockMFAStorage{}, &mockEmailVerifyHandler{}, &mockEmailSender{}, nil, nil, true, "Ziziphus", "development", nil, nil), authSvc, userRepo, sessMgr
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
		return &model.User{ID: id, Name: "Target User", Type: model.UserHuman, Discoverable: true}, nil
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
			{ID: "user_1", Name: "Alice", Type: model.UserHuman, Discoverable: true},
			{ID: "user_2", Name: "Bob", Type: model.UserHuman, Discoverable: true},
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

	var capturedID, capturedName, capturedAvatar, capturedCover, capturedEmail, capturedPrimaryColor, capturedSecondaryColor string
	var capturedDiscoverable, capturedAllowDirectChat bool
	userRepo.updateFunc = func(_ context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor string, discoverable, allowDirectChat bool) error {
		capturedID = id
		capturedName = name
		capturedAvatar = avatar
		capturedCover = cover
		capturedEmail = email
		capturedPrimaryColor = primaryColor
		capturedSecondaryColor = secondaryColor
		capturedDiscoverable = discoverable
		capturedAllowDirectChat = allowDirectChat
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
	if capturedCover != "" {
		t.Errorf("capturedCover = %q, want %q", capturedCover, "")
	}
	if capturedDiscoverable != true {
		t.Errorf("capturedDiscoverable = %v, want true", capturedDiscoverable)
	}
	if capturedAllowDirectChat != true {
		t.Errorf("capturedAllowDirectChat = %v, want true", capturedAllowDirectChat)
	}
	if capturedEmail != "" {
		t.Errorf("capturedEmail = %q, want %q", capturedEmail, "")
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
			createGroupFunc: func(_ context.Context, name, headline, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
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
		userGetter: &mockUserGetter{},
		idGen:      func() int64 { return 1 },
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
		userGetter: &mockUserGetter{},
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
		userGetter: &mockUserGetter{},
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
		userGetter: &mockUserGetter{},
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
		userGetter: &mockUserGetter{},
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
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
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
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
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

func TestMsgHandler_GetReceipts_Success(t *testing.T) {
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getFunc: func(_ context.Context, msgID int64) (*model.Message, error) {
				return &model.Message{MsgID: msgID, ConvID: "conv_1"}, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
		receipts: &mockReceiptStorage{
			getByMsgIDFunc: func(_ context.Context, msgID int64) ([]*model.Receipt, error) {
				return []*model.Receipt{
					{MsgID: 1, UserID: "user_2", Status: model.ReceiptRead, Timestamp: 1000},
					{MsgID: 1, UserID: "user_3", Status: model.ReceiptRead, Timestamp: 2000},
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/{msg_id}/receipts", nil)
	req = setChiURLParam(req, "msg_id", "1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetReceipts(w, req)

	resp := decodeResponse(t, w)
	items := resp.Data.([]interface{})
	if len(items) != 2 {
		t.Errorf("got %d receipts, want 2", len(items))
	}
	if items[0].(map[string]interface{})["user_id"] != "user_2" {
		t.Errorf("first user_id = %v, want user_2", items[0].(map[string]interface{})["user_id"])
	}
}

func TestMsgHandler_GetReceipts_InvalidID(t *testing.T) {
	handler := &MsgHandler{}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/{msg_id}/receipts", nil)
	req = setChiURLParam(req, "msg_id", "-1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetReceipts(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrBadMessage)
	}
}

func TestMsgHandler_GetReceipts_NotMember(t *testing.T) {
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getFunc: func(_ context.Context, msgID int64) (*model.Message, error) {
				return &model.Message{MsgID: msgID, ConvID: "conv_1"}, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return false, nil
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/{msg_id}/receipts", nil)
	req = setChiURLParam(req, "msg_id", "1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetReceipts(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

func TestMsgHandler_GetReceipts_Empty(t *testing.T) {
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getFunc: func(_ context.Context, msgID int64) (*model.Message, error) {
				return &model.Message{MsgID: msgID, ConvID: "conv_1"}, nil
			},
		},
		convMgr: &mockConvManager{
			isMemberFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return true, nil
			},
		},
		receipts: &mockReceiptStorage{
			getByMsgIDFunc: func(_ context.Context, msgID int64) ([]*model.Receipt, error) {
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/{msg_id}/receipts", nil)
	req = setChiURLParam(req, "msg_id", "999")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetReceipts(w, req)

	resp := decodeResponse(t, w)
	items := resp.Data.([]interface{})
	if len(items) != 0 {
		t.Errorf("got %d receipts, want 0", len(items))
	}
}

func TestMsgHandler_GetReceipts_MsgNotFound(t *testing.T) {
	handler := &MsgHandler{
		msgRepo: &mockMsgStorage{
			getFunc: func(_ context.Context, msgID int64) (*model.Message, error) {
				return nil, model.ErrConvNotFound
			},
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/messages/{msg_id}/receipts", nil)
	req = setChiURLParam(req, "msg_id", "999")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.GetReceipts(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

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
	callCount := 0
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			removeFunc: func(_ context.Context, userID, contactID string) error {
				// First call is the forward direction; capture those params.
				if callCount == 0 {
					capturedUserID = userID
					capturedContactID = contactID
				}
				callCount++
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
			requestJoinFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return false, nil
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
			requestJoinFunc: func(_ context.Context, convID, userID string) (bool, error) {
				return false, &model.AppError{Code: model.ErrBadMessage, Message: "join error", Key: "err.already_member"}
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

// =========================================================================
// Tests: ConvHandler – CreateP2P
// =========================================================================

func TestConvHandler_CreateP2P(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getOrCreateP2PFunc: func(_ context.Context, userA, userB string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: "p2p_1", Type: model.ConvP2P}, nil
			},
		},
		userGetter: &mockUserGetter{
			getByIDFunc: func(_ context.Context, id string) (*model.User, error) {
				return &model.User{ID: id, Name: "Partner", Type: model.UserHuman, AllowDirectChat: true}, nil
			},
		},
	}

	body := `{"user_id":"u2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/p2p", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateP2P(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	if data["conv_id"] != "p2p_1" {
		t.Errorf("conv_id = %v, want %q", data["conv_id"], "p2p_1")
	}
	if data["type"] != float64(model.ConvP2P) {
		t.Errorf("type = %v, want %v", data["type"], model.ConvP2P)
	}
}

func TestConvHandler_CreateP2P_EmptyUserID(t *testing.T) {
	handler := &ConvHandler{}
	body := `{"user_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/p2p", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateP2P(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestConvHandler_CreateP2P_Self(t *testing.T) {
	handler := &ConvHandler{}
	body := `{"user_id":"u1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/p2p", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateP2P(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestConvHandler_CreateP2P_DirectChatDisabled(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			areContactsFunc: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
		},
		userGetter: &mockUserGetter{
			getByIDFunc: func(_ context.Context, id string) (*model.User, error) {
				return &model.User{ID: id, Name: "Partner", Type: model.UserHuman, AllowDirectChat: false}, nil
			},
		},
	}

	body := `{"user_id":"u2"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/p2p", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateP2P(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

// =========================================================================
// Tests: ConvHandler – SearchGroups
// =========================================================================

func TestConvHandler_SearchGroups(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			searchByNameFunc: func(_ context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error) {
				return []*db.GroupSearchItem{{ConvID: "g1", Name: "Test Group"}}, 1, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/search?q=test&page=1&size=20", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.SearchGroups(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestConvHandler_SearchGroups_DefaultPagination(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			searchByNameFunc: func(_ context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error) {
				if page != 1 || size != 20 {
					t.Errorf("page=%d, size=%d want page=1 size=20", page, size)
				}
				return []*db.GroupSearchItem{}, 0, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/groups/search", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.SearchGroups(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

// =========================================================================
// Tests: ConvHandler – UnreadTotal
// =========================================================================

func TestConvHandler_UnreadTotal(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			getUserConvsFunc: func(_ context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error) {
				return []*db.ConvListItem{
					{ConvID: "c1", UnreadCount: 3},
					{ConvID: "c2", UnreadCount: 5},
				}, 2, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/unread/total", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.UnreadTotal(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["total"].(float64) != 8 {
		t.Errorf("total = %v, want 8", data["total"])
	}
}

// =========================================================================
// Tests: ConvHandler – Pin / Unpin
// =========================================================================

func TestConvHandler_Pin(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			pinFunc: func(_ context.Context, _, _ string) error { return nil },
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/pin", http.NoBody)
	req = setChiURLParam(req, "conv_id", "c1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Pin(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["pinned"] != true {
		t.Errorf("pinned = %v, want true", data["pinned"])
	}
}

func TestConvHandler_Unpin(t *testing.T) {
	handler := &ConvHandler{
		convRepo: &mockConvDataRepo{
			unpinFunc: func(_ context.Context, _, _ string) error { return nil },
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/unpin", http.NoBody)
	req = setChiURLParam(req, "conv_id", "c1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Unpin(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["pinned"] != false {
		t.Errorf("pinned = %v, want false", data["pinned"])
	}
}

// =========================================================================
// Tests: ConvHandler – Clone
// =========================================================================

func TestConvHandler_Clone(t *testing.T) {
	idGenCalls := 0
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: convID, Type: model.ConvGroup, Name: "Original"}, nil
			},
			getMemberRoleFunc: func(_ context.Context, _, _ string) (model.ConvRole, error) {
				return model.ConvRoleOwner, nil
			},
		},
		convRepo: &mockConvDataRepo{
			cloneFunc: func(_ context.Context, src, dst, owner, name string, idGen func() int64) error {
				return nil
			},
		},
		idGen: func() int64 {
			idGenCalls++
			return 100 + int64(idGenCalls)
		},
	}

	body := `{"name":"Cloned Group"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/clone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Clone(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	if data["name"] != "Cloned Group" {
		t.Errorf("name = %v, want %q", data["name"], "Cloned Group")
	}
}

func TestConvHandler_Clone_NotGroup(t *testing.T) {
	idGenCalls := 0
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: convID, Type: model.ConvP2P}, nil
			},
		},
		convRepo: &mockConvDataRepo{},
		idGen: func() int64 {
			idGenCalls++
			return 100 + int64(idGenCalls)
		},
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/clone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "p1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Clone(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestConvHandler_Clone_NotOwner(t *testing.T) {
	idGenCalls := 0
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			getFunc: func(_ context.Context, convID string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: convID, Type: model.ConvGroup}, nil
			},
			getMemberRoleFunc: func(_ context.Context, _, _ string) (model.ConvRole, error) {
				return model.ConvRoleMember, nil
			},
		},
		idGen: func() int64 {
			idGenCalls++
			return 100 + int64(idGenCalls)
		},
	}

	body := `{}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/conversations/{conv_id}/clone", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Clone(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

// =========================================================================
// Tests: ContactHandler – RequestContact
// =========================================================================

func TestContactHandler_RequestContact_Success(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			existsAnyDirectionFunc: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			getByPairFunc: func(_ context.Context, _, _ string) (*model.ContactRequest, error) {
				return nil, nil
			},
			insertFunc: func(_ context.Context, req *model.ContactRequest) (int64, error) {
				return 42, nil
			},
			updateFormMsgIDFunc: func(_ context.Context, _, _ int64) error {
				return nil
			},
		},
		userRepo: &mockUserQueryRepo{
			getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
				return map[string]*model.User{
					ids[0]: {ID: ids[0], Name: "TargetUser"},
				}, nil
			},
		},
		ingest: &mockFormMessageSender{
			sendFormMessageFunc: func(_ context.Context, _ string, _ *model.FormDefinitionBody) (*model.Message, error) {
				return &model.Message{MsgID: 10}, nil
			},
			sendSystemMessageFunc: func(_ context.Context, _, _ string, _ ...string) (*model.Message, error) {
				return &model.Message{MsgID: 20}, nil
			},
		},
		convMgr: &mockSystemConvManager{
			getOrCreateSystemConvFunc: func(_ context.Context, _ string) (*model.Conversation, error) {
				return &model.Conversation{ConvID: "sys:test"}, nil
			},
		},
	}

	body := `{"user_id":"target1","message":"Hello!"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	if data["request_id"].(float64) != 42 {
		t.Errorf("request_id = %v, want 42", data["request_id"])
	}
	if data["form_msg_id"].(float64) != 10 {
		t.Errorf("form_msg_id = %v, want 10", data["form_msg_id"])
	}
}

func TestContactHandler_RequestContact_EmptyTarget(t *testing.T) {
	handler := &ContactHandler{}

	body := `{"user_id":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestContactHandler_RequestContact_Self(t *testing.T) {
	handler := &ContactHandler{}

	body := `{"user_id":"sender1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrContactRequestSelf.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrContactRequestSelf.Code)
	}
}

func TestContactHandler_RequestContact_UserNotFound(t *testing.T) {
	handler := &ContactHandler{
		userRepo: &mockUserQueryRepo{
			getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
				return map[string]*model.User{}, nil
			},
		},
	}

	body := `{"user_id":"nonexistent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

func TestContactHandler_RequestContact_AlreadyFriends(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			existsAnyDirectionFunc: func(_ context.Context, _, _ string) (bool, error) {
				return true, nil
			},
		},
		userRepo: &mockUserQueryRepo{
			getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
				return map[string]*model.User{
					ids[0]: {ID: ids[0], Name: "ExistingFriend"},
				}, nil
			},
		},
	}

	body := `{"user_id":"friend1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrAlreadyFriends.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrAlreadyFriends.Code)
	}
}

func TestContactHandler_RequestContact_PendingRequest(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			existsAnyDirectionFunc: func(_ context.Context, _, _ string) (bool, error) {
				return false, nil
			},
			getByPairFunc: func(_ context.Context, _, _ string) (*model.ContactRequest, error) {
				return &model.ContactRequest{ID: 1, Status: model.ContactRequestPending}, nil
			},
		},
		userRepo: &mockUserQueryRepo{
			getByIDsFunc: func(_ context.Context, ids []string) (map[string]*model.User, error) {
				return map[string]*model.User{
					ids[0]: {ID: ids[0], Name: "TargetUser"},
				}, nil
			},
		},
	}

	body := `{"user_id":"target1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/contact-requests", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "sender1")
	w := httptest.NewRecorder()
	handler.RequestContact(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrContactRequestDuplicate.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrContactRequestDuplicate.Code)
	}
}

// =========================================================================
// Tests: ContactHandler – ListSentRequests
// =========================================================================

func TestContactHandler_ListSentRequests(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			listSentFunc: func(_ context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error) {
				return []*model.ContactRequest{
					{ID: 1, FromUserID: userID, ToUserID: "u2", Status: model.ContactRequestPending},
				}, 1, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/sent?page=1&size=20", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListSentRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestContactHandler_ListSentRequests_DefaultPagination(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			listSentFunc: func(_ context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error) {
				if page != 1 || size != 20 {
					t.Errorf("got page=%d size=%d, want page=1 size=20", page, size)
				}
				return []*model.ContactRequest{}, 0, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/sent", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListSentRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

// =========================================================================
// Tests: ContactHandler – ListReceivedRequests
// =========================================================================

func TestContactHandler_ListReceivedRequests(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			listReceivedFunc: func(_ context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error) {
				return []*model.ContactRequest{
					{ID: 1, FromUserID: "u2", ToUserID: userID, Status: model.ContactRequestPending},
				}, 1, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/received?page=1&size=20", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListReceivedRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestContactHandler_ListReceivedRequests_WithStatus(t *testing.T) {
	var capturedStatus int
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			listReceivedFunc: func(_ context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error) {
				capturedStatus = status
				return []*model.ContactRequest{}, 0, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/received?page=1&size=20&status=1", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListReceivedRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	if capturedStatus != 1 {
		t.Errorf("status = %d, want 1", capturedStatus)
	}
}

// =========================================================================
// Tests: ContactHandler – GetRequestByFormMsgID
// =========================================================================

func TestContactHandler_GetRequestByFormMsgID(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			getByFormMsgIDFunc: func(_ context.Context, msgID int64) (*model.ContactRequest, error) {
				return &model.ContactRequest{ID: 1, FromUserID: "u1", ToUserID: "u2", FormMsgID: msgID}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/by-form/{msg_id}", nil)
	req = setChiURLParam(req, "msg_id", "100")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.GetRequestByFormMsgID(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestContactHandler_GetRequestByFormMsgID_NotFound(t *testing.T) {
	handler := &ContactHandler{
		reqRepo: &mockContactRequestStorage{
			getByFormMsgIDFunc: func(_ context.Context, msgID int64) (*model.ContactRequest, error) {
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/by-form/{msg_id}", nil)
	req = setChiURLParam(req, "msg_id", "999")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.GetRequestByFormMsgID(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrContactRequestNotFound.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrContactRequestNotFound.Code)
	}
}

func TestContactHandler_GetRequestByFormMsgID_InvalidID(t *testing.T) {
	handler := &ContactHandler{}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contact-requests/by-form/{msg_id}", nil)
	req = setChiURLParam(req, "msg_id", "invalid")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.GetRequestByFormMsgID(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// Tests: FileHandler – GetInfo
// =========================================================================

func TestFileHandler_GetInfo(t *testing.T) {
	handler := &FileHandler{
		fileDB: &mockFileDB{
			getByIDFunc: func(_ context.Context, fileID string) (*model.FileInfo, error) {
				return &model.FileInfo{
					FileID: fileID,
					URL:    "http://example.com/file.txt",
					Size:   1024,
					Name:   "test.txt",
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/{file_id}", nil)
	req = setChiURLParam(req, "file_id", "file_1")
	w := httptest.NewRecorder()
	handler.GetInfo(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["file_id"] != "file_1" {
		t.Errorf("file_id = %v, want %q", data["file_id"], "file_1")
	}
}

func TestFileHandler_GetInfo_NotFound(t *testing.T) {
	handler := &FileHandler{
		fileDB: &mockFileDB{
			getByIDFunc: func(_ context.Context, fileID string) (*model.FileInfo, error) {
				return nil, fmt.Errorf("not found")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/files/{file_id}", nil)
	req = setChiURLParam(req, "file_id", "nonexistent")
	w := httptest.NewRecorder()
	handler.GetInfo(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// =========================================================================
// Tests: FileHandler – pure functions
// =========================================================================

func TestFileHandler_contentTypeByExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".png", "image/png"},
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".gif", "image/gif"},
		{".webp", "image/webp"},
		{".mp4", "video/mp4"},
		{".mp3", "audio/mpeg"},
		{".pdf", "application/pdf"},
		{".unknown", "application/octet-stream"},
	}
	for _, tc := range tests {
		got := contentTypeByExt(tc.ext)
		if got != tc.want {
			t.Errorf("contentTypeByExt(%q) = %q, want %q", tc.ext, got, tc.want)
		}
	}
}

func TestFileHandler_isImageExt(t *testing.T) {
	tests := []struct {
		ext  string
		want bool
	}{
		{".png", true},
		{".jpg", true},
		{".jpeg", true},
		{".gif", true},
		{".webp", false},
		{".mp4", false},
		{".txt", false},
	}
	for _, tc := range tests {
		got := isImageExt(tc.ext)
		if got != tc.want {
			t.Errorf("isImageExt(%q) = %v, want %v", tc.ext, got, tc.want)
		}
	}
}

func TestFileHandler_decodeImageDimensions(t *testing.T) {
	// Create a small 2x2 red PNG
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	png.Encode(&buf, img)

	w, h, err := decodeImageDimensions(buf.Bytes())
	if err != nil {
		t.Fatalf("decodeImageDimensions error: %v", err)
	}
	if w != 2 || h != 2 {
		t.Errorf("got %dx%d, want 2x2", w, h)
	}
}

func TestFileHandler_decodeImageDimensions_Invalid(t *testing.T) {
	_, _, err := decodeImageDimensions([]byte("not-an-image"))
	if err == nil {
		t.Error("expected error for invalid image data, got nil")
	}
}

func TestFileHandler_resizeImage(t *testing.T) {
	// Create a small 4x4 red PNG
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	png.Encode(&buf, img)

	// Resize to 2x2
	result, ct, err := resizeImage(buf.Bytes(), 2, 2, ".png", false)
	if err != nil {
		t.Fatalf("resizeImage error: %v", err)
	}
	if len(result) == 0 {
		t.Error("resized image is empty")
	}
	if ct != "image/png" {
		t.Errorf("content type = %q, want %q", ct, "image/png")
	}
}

func TestFileHandler_resizeImage_JPG(t *testing.T) {
	var buf bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	png.Encode(&buf, img)

	// Resize with only width
	result, ct, err := resizeImage(buf.Bytes(), 2, 0, ".jpg", false)
	if err != nil {
		t.Fatalf("resizeImage error: %v", err)
	}
	if len(result) == 0 {
		t.Error("resized image is empty")
	}
	if ct != "image/jpeg" {
		t.Errorf("content type = %q, want %q", ct, "image/jpeg")
	}
}

// =========================================================================
// Tests: FileHandler – NewFileHandler
// =========================================================================

func TestNewFileHandler(t *testing.T) {
	h := NewFileHandler(nil, nil, nil, "http://example.com", nil, nil, nil, nil)
	if h == nil {
		t.Fatal("NewFileHandler returned nil")
	}
	if h.baseURL != "http://example.com" {
		t.Errorf("baseURL = %q, want %q", h.baseURL, "http://example.com")
	}
}

// =========================================================================
// Tests: SessionHandler
// =========================================================================

func TestNewSessionHandler(t *testing.T) {
	h := NewSessionHandler(nil, nil)
	if h == nil {
		t.Fatal("NewSessionHandler returned nil")
	}
}

func TestSessionHandler_ListSessions(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getUserSessionIDsFunc: func(_ context.Context, userID string) []string {
				return []string{"s1", "s2"}
			},
			getFunc: func(_ context.Context, sessionID string) *model.Session {
				return &model.Session{SessionID: sessionID, UserID: "u1"}
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListSessions(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	sessions, ok := resp.Data.([]interface{})
	if !ok {
		t.Fatal("expected array in data")
	}
	if len(sessions) != 2 {
		t.Errorf("got %d sessions, want 2", len(sessions))
	}
}

func TestSessionHandler_ListSessions_Empty(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getUserSessionIDsFunc: func(_ context.Context, userID string) []string {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/sessions", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListSessions(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

// =========================================================================
// Tests: UserHandler – Agents
// =========================================================================

func TestUserHandler_ListMyAgents(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			listAgentsFunc: func(_ context.Context, uid string) ([]*model.User, error) {
				return []*model.User{
					{ID: "agent1", Name: "Agent 1"},
				}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/agents", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListMyAgents(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestUserHandler_ListMyAgents_NilList(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			listAgentsFunc: func(_ context.Context, uid string) ([]*model.User, error) {
				return nil, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/users/me/agents", nil)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.ListMyAgents(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestUserHandler_CreateAgent(t *testing.T) {
	idGenCalls := 0
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			countAgentsFunc: func(_ context.Context, uid string) (int, error) {
				return 0, nil
			},
			createFunc: func(_ context.Context, u *model.User) error {
				return nil
			},
		},
		idGen: func() int64 {
			idGenCalls++
			return 100 + int64(idGenCalls)
		},
	}

	body := `{"name":"My Agent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateAgent(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
}

func TestUserHandler_CreateAgent_EmptyName(t *testing.T) {
	handler := &UserHandler{}

	body := `{"name":""}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateAgent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_CreateAgent_LimitReached(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			countAgentsFunc: func(_ context.Context, uid string) (int, error) {
				return 10, nil
			},
		},
	}

	body := `{"name":"Another Agent"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/me/agents", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.CreateAgent(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrBadMessage {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrBadMessage)
	}
}

func TestUserHandler_DeleteAgent(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			deleteAgentFunc: func(_ context.Context, agentID, uid string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/agents/{agent_id}", http.NoBody)
	req = setChiURLParam(req, "agent_id", "agent1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteAgent(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
}

func TestUserHandler_DeleteAgent_EmptyID(t *testing.T) {
	handler := &UserHandler{}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me/agents/", http.NoBody)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteAgent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_DeleteAccount(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			deleteAccountFunc: func(_ context.Context, userID string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", http.NoBody)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteAccount(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0", resp.Code)
	}
	data := resp.Data.(map[string]interface{})
	if data["user_id"] != "u1" {
		t.Errorf("user_id = %v, want %q", data["user_id"], "u1")
	}
}

func TestUserHandler_DeleteAccount_Error(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			deleteAccountFunc: func(_ context.Context, userID string) error {
				return fmt.Errorf("db error")
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/me", http.NoBody)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteAccount(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrInternalServer.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrInternalServer.Code)
	}
}

func TestUserHandler_UpdateAgent(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			updateAgentFunc: func(_ context.Context, agentID, uid, name, avatar, cover, primaryColor, secondaryColor string, wakeMode model.WakeMode, discoverable, allowDirectChat bool) error {
				return nil
			},
		},
	}

	body := `{"name":"Updated Agent"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/agents/{agent_id}", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setChiURLParam(req, "agent_id", "agent1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.UpdateAgent(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
}

func TestUserHandler_UpdateAgent_EmptyID(t *testing.T) {
	handler := &UserHandler{}

	body := `{"name":"Updated"}`
	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/agents/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.UpdateAgent(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_RegenerateAgentKey(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			listAgentsFunc: func(_ context.Context, uid string) ([]*model.User, error) {
				return []*model.User{{ID: "agent1"}, {ID: "agent2"}}, nil
			},
			updateAgentAPIKeyFunc: func(_ context.Context, agentID, uid, apiKey string) error {
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/agents/{agent_id}/regenerate-key", http.NoBody)
	req = setChiURLParam(req, "agent_id", "agent1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.RegenerateAgentKey(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	key, ok := data["api_key"].(string)
	if !ok || key == "" {
		t.Errorf("api_key = %q, want non-empty", key)
	}
	if !strings.HasPrefix(key, "sk-") {
		t.Errorf("api_key = %q, want sk- prefix", key)
	}
}

func TestUserHandler_RegenerateAgentKey_EmptyID(t *testing.T) {
	handler := &UserHandler{}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/agents//regenerate-key", http.NoBody)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.RegenerateAgentKey(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestUserHandler_RegenerateAgentKey_NotFound(t *testing.T) {
	handler := &UserHandler{
		userRepo: &mockUserRepo{
			listAgentsFunc: func(_ context.Context, uid string) ([]*model.User, error) {
				return []*model.User{{ID: "other_agent"}}, nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPut, "/api/v1/users/me/agents/{agent_id}/regenerate-key", http.NoBody)
	req = setChiURLParam(req, "agent_id", "nonexistent")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.RegenerateAgentKey(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

// =========================================================================
// Tests: SessionHandler – DeleteSession
// =========================================================================

func TestSessionHandler_DeleteSession_EmptyID(t *testing.T) {
	handler := &SessionHandler{
		gwMgr: gateway.NewManager(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/", http.NoBody)
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteSession(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestSessionHandler_DeleteSession_NotFound(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getFunc: func(_ context.Context, sessionID string) *model.Session {
				return nil
			},
		},
		gwMgr: gateway.NewManager(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/{session_id}", http.NoBody)
	req = setChiURLParam(req, "session_id", "nonexistent")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteSession(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

func TestSessionHandler_DeleteSession_WrongUser(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getFunc: func(_ context.Context, sessionID string) *model.Session {
				return &model.Session{SessionID: sessionID, UserID: "other_user"}
			},
		},
		gwMgr: gateway.NewManager(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/{session_id}", http.NoBody)
	req = setChiURLParam(req, "session_id", "s1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteSession(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNotFound {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNotFound)
	}
}

func TestSessionHandler_DeleteSession_Success(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getFunc: func(_ context.Context, sessionID string) *model.Session {
				return &model.Session{SessionID: sessionID, UserID: "u1"}
			},
			deleteFunc: func(_ context.Context, sessionID string) error {
				return nil
			},
		},
		gwMgr: gateway.NewManager(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/{session_id}", http.NoBody)
	req = setChiURLParam(req, "session_id", "s1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteSession(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != 0 {
		t.Fatalf("code = %d, want 0: %s", resp.Code, resp.Msg)
	}
	data := resp.Data.(map[string]interface{})
	if data["session_id"] != "s1" {
		t.Errorf("session_id = %v, want %q", data["session_id"], "s1")
	}
}

func TestSessionHandler_DeleteSession_DeleteError(t *testing.T) {
	handler := &SessionHandler{
		sessMgr: &mockSessionManager{
			getFunc: func(_ context.Context, sessionID string) *model.Session {
				return &model.Session{SessionID: sessionID, UserID: "u1"}
			},
			deleteFunc: func(_ context.Context, sessionID string) error {
				return fmt.Errorf("delete failed")
			},
		},
		gwMgr: gateway.NewManager(),
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/sessions/{session_id}", http.NoBody)
	req = setChiURLParam(req, "session_id", "s1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.DeleteSession(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrInternalServer.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrInternalServer.Code)
	}
}

// =========================================================================
// Tests: ConvHandler – JoinRequest error paths
// =========================================================================

func TestConvHandler_ListJoinRequests_AppError(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			listJoinRequestsFunc: func(_ context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
				return nil, &model.AppError{Code: model.ErrNoPermission, Message: "forbidden"}
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.ListJoinRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestConvHandler_ListJoinRequests_GenericError(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			listJoinRequestsFunc: func(_ context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
				return nil, fmt.Errorf("db error")
			},
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/conversations/{conv_id}/join-requests", http.NoBody)
	req = setChiURLParam(req, "conv_id", "g1")
	req = setAuthCtx(req, "user_1")
	w := httptest.NewRecorder()
	handler.ListJoinRequests(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrInternalServer.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrInternalServer.Code)
	}
}

func TestConvHandler_ApproveJoinRequest_AppError(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			approveJoinRequestFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return &model.AppError{Code: model.ErrNoPermission, Message: "forbidden"}
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
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

func TestConvHandler_RejectJoinRequest_AppError(t *testing.T) {
	handler := &ConvHandler{
		convMgr: &mockConvManager{
			rejectJoinRequestFunc: func(_ context.Context, convID, userID, operatorID string) error {
				return &model.AppError{Code: model.ErrNoPermission, Message: "forbidden"}
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
	if resp.Code != model.ErrNoPermission {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrNoPermission)
	}
}

func TestContactHandler_Remove_Error(t *testing.T) {
	handler := &ContactHandler{
		contactRepo: &mockContactStorage{
			removeFunc: func(_ context.Context, userID, contactID string) error {
				return fmt.Errorf("db error")
			},
		},
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/contacts/{user_id}", http.NoBody)
	req = setChiURLParam(req, "user_id", "contact1")
	req = setAuthCtx(req, "u1")
	w := httptest.NewRecorder()
	handler.Remove(w, req)

	resp := decodeResponse(t, w)
	if resp.Code != model.ErrInternalServer.Code {
		t.Errorf("code = %d, want %d", resp.Code, model.ErrInternalServer.Code)
	}
}

func TestRegister_PasswordMinLength(t *testing.T) {
	// Password < 8 chars should be rejected.
	userRepo := &mockUserRepo{
		createFunc: func(_ context.Context, _ *model.User) error { return nil },
	}
	handler := &UserHandler{userRepo: userRepo, allowRegistration: true}

	reqBody := `{"account":"test","name":"Test","password":"1234567"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/users/register", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.Register(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Register with 7-char password: got status %d; want 400", w.Code)
	}
}
