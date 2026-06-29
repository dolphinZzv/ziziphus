package conversation

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/model"
)

// ---------------------------------------------------------------------------
// Mock convRepo
// ---------------------------------------------------------------------------

type mockConvRepo struct {
	mu      sync.Mutex
	convs   map[string]*model.Conversation
	members map[string]map[string]*model.ConvMember // convID -> userID -> member
}

func newMockConvRepo() *mockConvRepo {
	return &mockConvRepo{
		convs:   make(map[string]*model.Conversation),
		members: make(map[string]map[string]*model.ConvMember),
	}
}

func (r *mockConvRepo) Create(_ context.Context, c *model.Conversation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.convs[c.ConvID]; exists {
		return fmt.Errorf("conversation already exists: %s", c.ConvID)
	}
	cp := *c
	r.convs[c.ConvID] = &cp
	return nil
}

func (r *mockConvRepo) CreateTx(_ context.Context, _ pgx.Tx, c *model.Conversation) error {
	return r.Create(context.Background(), c)
}

func (r *mockConvRepo) Get(_ context.Context, convID string) (*model.Conversation, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.convs[convID]
	if !ok {
		return nil, model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	cp := *c
	return &cp, nil
}

func (r *mockConvRepo) UpdateLastMsg(_ context.Context, convID string, msgID int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.convs[convID]
	if !ok {
		return model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	c.LastMsgID = msgID
	return nil
}

func (r *mockConvRepo) AddMember(_ context.Context, convID, userID string, role model.ConvRole) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.convs[convID]; !ok {
		return model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	if r.members[convID] == nil {
		r.members[convID] = make(map[string]*model.ConvMember)
	}
	r.members[convID][userID] = &model.ConvMember{
		ConvID: convID,
		UserID: userID,
		Role:   role,
	}
	return nil
}

func (r *mockConvRepo) AddMemberTx(_ context.Context, _ pgx.Tx, convID, userID string, role model.ConvRole) error {
	return r.AddMember(context.Background(), convID, userID, role)
}

func (r *mockConvRepo) RemoveMember(_ context.Context, convID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.convs[convID]; !ok {
		return model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	delete(r.members[convID], userID)
	return nil
}

func (r *mockConvRepo) GetMembers(_ context.Context, convID string) ([]*model.ConvMember, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.convs[convID]; !ok {
		return nil, model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	var out []*model.ConvMember
	for _, m := range r.members[convID] {
		cp := *m
		out = append(out, &cp)
	}
	return out, nil
}

func (r *mockConvRepo) IsDirectChatBlocked(_ context.Context, _ string) (bool, error) { return false, nil }

func (r *mockConvRepo) IsMember(_ context.Context, convID, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.convs[convID]; !ok {
		return false, model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	_, ok := r.members[convID][userID]
	return ok, nil
}

func (r *mockConvRepo) GetMemberRole(_ context.Context, convID, userID string) (model.ConvRole, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.convs[convID]; !ok {
		return 0, model.NewAppError(model.ErrNotFound, "会话不存在")
	}
	m, ok := r.members[convID][userID]
	if !ok {
		return 0, model.NewAppError(model.ErrNotFound, "成员不存在")
	}
	return m.Role, nil
}

// ---------------------------------------------------------------------------
// Mock userRepo
// ---------------------------------------------------------------------------

type mockUserRepo struct {
	mu    sync.Mutex
	users map[string]*model.User
}

func newMockUserRepo() *mockUserRepo {
	return &mockUserRepo{users: make(map[string]*model.User)}
}

func (r *mockUserRepo) addUser(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[id] = &model.User{ID: id, Name: id, Account: id}
}

func (r *mockUserRepo) GetByID(_ context.Context, id string) (*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.users[id]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", id)
	}
	return u, nil
}

func (r *mockUserRepo) GetByIDs(_ context.Context, ids []string) (map[string]*model.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	result := make(map[string]*model.User, len(ids))
	for _, id := range ids {
		if u, ok := r.users[id]; ok {
			result[id] = u
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Mock msgRepo
// ---------------------------------------------------------------------------

type mockMsgRepo struct{}

func (m *mockMsgRepo) GetMaxConvSeq(_ context.Context, convID string) (int64, error) {
	return 0, nil
}

// ---------------------------------------------------------------------------
// Mock seqCache
// ---------------------------------------------------------------------------

type mockSeqCache struct {
	mu   sync.Mutex
	seqs map[string]int64
}

func newMockSeqCache() *mockSeqCache {
	return &mockSeqCache{seqs: make(map[string]int64)}
}

func (s *mockSeqCache) InitConvSeq(_ context.Context, convID string, seq int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqs[convID] = seq
	return nil
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func newManager() (*Manager, *mockConvRepo, *mockUserRepo, *mockSeqCache) {
	convRepo := newMockConvRepo()
	msgRepo := &mockMsgRepo{}
	userRepo := newMockUserRepo()
	seqCache := newMockSeqCache()
	return NewManager(convRepo, msgRepo, seqCache, userRepo, nil), convRepo, userRepo, seqCache
}

func counterIDGen() func() int64 {
	var i int64
	return func() int64 {
		i++
		return i
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestGetOrCreateP2P_ReturnsExisting(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	// Pre-seed a conversation via the repo
	userA, userB := "user_a", "user_b"
	convID := model.MakeP2PConvID(userA, userB)
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvP2P, Name: ""})

	got, err := mgr.GetOrCreateP2P(ctx, userA, userB)
	if err != nil {
		t.Fatalf("GetOrCreateP2P returned error: %v", err)
	}
	if got.ConvID != convID {
		t.Fatalf("expected convID %q, got %q", convID, got.ConvID)
	}
}

func TestGetOrCreateP2P_CreatesNewWithMembers(t *testing.T) {
	mgr, _, _, seqCache := newManager()
	ctx := context.Background()

	userA, userB := "user_a", "user_b"
	got, err := mgr.GetOrCreateP2P(ctx, userA, userB)
	if err != nil {
		t.Fatalf("GetOrCreateP2P returned error: %v", err)
	}

	expectedID := model.MakeP2PConvID(userA, userB)
	if got.ConvID != expectedID {
		t.Fatalf("expected convID %q, got %q", expectedID, got.ConvID)
	}
	if got.Type != model.ConvP2P {
		t.Fatalf("expected ConvP2P type, got %v", got.Type)
	}
	if got.OwnerID != userA {
		t.Fatalf("expected owner %q, got %q", userA, got.OwnerID)
	}

	// Verify both members exist
	isMemberA, err := mgr.IsMember(ctx, got.ConvID, userA)
	if err != nil {
		t.Fatalf("IsMember(%s) error: %v", userA, err)
	}
	if !isMemberA {
		t.Errorf("expected userA to be a member")
	}

	isMemberB, err := mgr.IsMember(ctx, got.ConvID, userB)
	if err != nil {
		t.Fatalf("IsMember(%s) error: %v", userB, err)
	}
	if !isMemberB {
		t.Errorf("expected userB to be a member")
	}

	// Verify seq was initialized
	seqCache.mu.Lock()
	seq, ok := seqCache.seqs[expectedID]
	seqCache.mu.Unlock()
	if !ok {
		t.Errorf("expected conv seq to be initialized")
	} else if seq != 0 {
		t.Errorf("expected seq 0, got %d", seq)
	}

	// Verify roles
	roleA, err := mgr.convRepo.GetMemberRole(ctx, got.ConvID, userA)
	if err != nil {
		t.Fatalf("GetMemberRole(userA) error: %v", err)
	}
	if roleA != model.ConvRoleOwner {
		t.Errorf("expected userA role Owner, got %v", roleA)
	}

	roleB, err := mgr.convRepo.GetMemberRole(ctx, got.ConvID, userB)
	if err != nil {
		t.Fatalf("GetMemberRole(userB) error: %v", err)
	}
	if roleB != model.ConvRoleMember {
		t.Errorf("expected userB role Member, got %v", roleB)
	}
}

func TestCreateGroup_CreatesGroupWithOwnerAndMembers(t *testing.T) {
	mgr, _, userRepo, seqCache := newManager()
	ctx := context.Background()

	ownerID := "owner"
	memberIDs := []string{"m1", "m2", "m3"}
	name := "test group"
	gen := counterIDGen()
	for _, u := range append(memberIDs, ownerID) {
		userRepo.addUser(u)
	}

	got, err := mgr.CreateGroup(ctx, name, ownerID, memberIDs, gen)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	expectedID := fmt.Sprintf("%s%d", model.GroupConvIDPrefix, 1)
	if got.ConvID != expectedID {
		t.Fatalf("expected convID %q, got %q", expectedID, got.ConvID)
	}
	if got.Type != model.ConvGroup {
		t.Fatalf("expected ConvGroup type, got %v", got.Type)
	}
	if got.Name != name {
		t.Fatalf("expected name %q, got %q", name, got.Name)
	}
	if got.OwnerID != ownerID {
		t.Fatalf("expected owner %q, got %q", ownerID, got.OwnerID)
	}
	if got.MaxMembers != 200 {
		t.Fatalf("expected MaxMembers 200, got %d", got.MaxMembers)
	}

	// Verify owner role
	roleOwner, err := mgr.convRepo.GetMemberRole(ctx, got.ConvID, ownerID)
	if err != nil {
		t.Fatalf("GetMemberRole(owner) error: %v", err)
	}
	if roleOwner != model.ConvRoleOwner {
		t.Errorf("expected owner role Owner, got %v", roleOwner)
	}

	// Verify all members exist with correct role
	for _, mid := range memberIDs {
		ok, err := mgr.IsMember(ctx, got.ConvID, mid)
		if err != nil {
			t.Fatalf("IsMember(%s) error: %v", mid, err)
		}
		if !ok {
			t.Errorf("expected member %s to be in the conversation", mid)
		}
	}

	roleM1, err := mgr.convRepo.GetMemberRole(ctx, got.ConvID, "m1")
	if err != nil {
		t.Fatalf("GetMemberRole(m1) error: %v", err)
	}
	if roleM1 != model.ConvRoleMember {
		t.Errorf("expected m1 role Member, got %v", roleM1)
	}

	// Verify seq was initialized
	seqCache.mu.Lock()
	seq, ok := seqCache.seqs[expectedID]
	seqCache.mu.Unlock()
	if !ok {
		t.Errorf("expected conv seq to be initialized")
	} else if seq != 0 {
		t.Errorf("expected seq 0, got %d", seq)
	}
}

func TestCreateGroup_DedupesOwnerFromMemberList(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	ownerID := "owner"
	memberIDs := []string{"m1", "owner", "m2"} // owner appears in member list
	gen := counterIDGen()
	for _, u := range []string{ownerID, "m1", "m2"} {
		userRepo.addUser(u)
	}

	got, err := mgr.CreateGroup(ctx, "dedup", ownerID, memberIDs, gen)
	if err != nil {
		t.Fatalf("CreateGroup returned error: %v", err)
	}

	// Owner role must be Owner
	role, err := convRepo.GetMemberRole(ctx, got.ConvID, ownerID)
	if err != nil {
		t.Fatalf("GetMemberRole(owner) error: %v", err)
	}
	if role != model.ConvRoleOwner {
		t.Errorf("expected owner role Owner, got %v", role)
	}

	// There should be exactly 3 distinct members: owner + m1 + m2
	members, err := mgr.GetMembers(ctx, got.ConvID)
	if err != nil {
		t.Fatalf("GetMembers error: %v", err)
	}
	if len(members) != 3 {
		t.Errorf("expected 3 members (owner deduped), got %d", len(members))
	}

	// Verify all expected members exist
	for _, uid := range []string{ownerID, "m1", "m2"} {
		ok, err := mgr.IsMember(ctx, got.ConvID, uid)
		if err != nil {
			t.Fatalf("IsMember(%s) error: %v", uid, err)
		}
		if !ok {
			t.Errorf("expected member %s to be present", uid)
		}
	}
}

func TestGet_ReturnsConversation(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	conv := &model.Conversation{
		ConvID: "conv1",
		Type:   model.ConvGroup,
		Name:   "test",
	}
	if err := convRepo.Create(ctx, conv); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := mgr.Get(ctx, "conv1")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if got.ConvID != "conv1" {
		t.Errorf("expected ConvID conv1, got %s", got.ConvID)
	}
}

func TestGet_NonExistent(t *testing.T) {
	mgr, _, _, _ := newManager()
	ctx := context.Background()

	_, err := mgr.Get(ctx, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent conversation")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNotFound {
		t.Errorf("expected ErrNotFound code, got %d", appErr.Code)
	}
}

func TestAddMember_AdminSucceeds(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	// Register users
	for _, u := range []string{"admin", "existing", "new_user"} {
		userRepo.addUser(u)
	}
	// Set up a conversation with an admin
	convID := "conv_admin_add"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "admin", model.ConvRoleAdmin)
	_ = convRepo.AddMember(ctx, convID, "existing", model.ConvRoleMember)

	err := mgr.AddMember(ctx, convID, "new_user", "admin")
	if err != nil {
		t.Fatalf("AddMember by admin returned error: %v", err)
	}

	ok, err := mgr.IsMember(ctx, convID, "new_user")
	if err != nil {
		t.Fatalf("IsMember error: %v", err)
	}
	if !ok {
		t.Errorf("expected new_user to be a member after add")
	}

	// Verify role is Member
	role, err := convRepo.GetMemberRole(ctx, convID, "new_user")
	if err != nil {
		t.Fatalf("GetMemberRole error: %v", err)
	}
	if role != model.ConvRoleMember {
		t.Errorf("expected new_user role Member, got %v", role)
	}
}

func TestAddMember_MemberNoPermission(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	userRepo.addUser("regular")
	userRepo.addUser("victim")
	convID := "conv_no_perm"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "regular", model.ConvRoleMember)

	err := mgr.AddMember(ctx, convID, "victim", "regular")
	if err == nil {
		t.Fatal("expected error when member tries to add another user")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNoPermission {
		t.Errorf("expected ErrNoPermission code, got %d", appErr.Code)
	}
}

func TestAddMember_NonExistentConv(t *testing.T) {
	mgr, _, _, _ := newManager()
	ctx := context.Background()

	err := mgr.AddMember(ctx, "ghost_conv", "someone", "anyone")
	if err == nil {
		t.Fatal("expected error for non-existent conversation")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNotFound {
		t.Errorf("expected ErrNotFound code, got %d", appErr.Code)
	}
}

func TestAddMember_UserNotFound(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	convID := "conv_user_not_found"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	userRepo.addUser("admin")
	_ = convRepo.AddMember(ctx, convID, "admin", model.ConvRoleAdmin)

	err := mgr.AddMember(ctx, convID, "nonexistent_user", "admin")
	if err == nil {
		t.Fatal("expected error when target user does not exist")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNotFound {
		t.Errorf("expected ErrNotFound code, got %d", appErr.Code)
	}
}

func TestAddMember_NotGroupConv(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	// Create a P2P conversation
	convID := "user_a:user_b"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvP2P})
	userRepo.addUser("user_a")
	userRepo.addUser("user_b")
	_ = convRepo.AddMember(ctx, convID, "user_a", model.ConvRoleOwner)
	_ = convRepo.AddMember(ctx, convID, "user_b", model.ConvRoleMember)

	err := mgr.AddMember(ctx, convID, "user_c", "user_a")
	if err == nil {
		t.Fatal("expected error when adding member to P2P conversation")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrBadMessage {
		t.Errorf("expected ErrBadMessage code, got %d", appErr.Code)
	}
}

func TestAddMember_GroupFull(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManager()
	ctx := context.Background()

	convID := "conv_full"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup, MaxMembers: 2})
	userRepo.addUser("owner")
	userRepo.addUser("existing")
	userRepo.addUser("new_user")
	_ = convRepo.AddMember(ctx, convID, "owner", model.ConvRoleOwner)
	_ = convRepo.AddMember(ctx, convID, "existing", model.ConvRoleMember)

	// group already has 2 members, max is 2
	err := mgr.AddMember(ctx, convID, "new_user", "owner")
	if err == nil {
		t.Fatal("expected error when group is full")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrTooLarge {
		t.Errorf("expected ErrTooLarge code, got %d", appErr.Code)
	}
}

func TestRemoveMember_ByAdminSucceeds(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_admin_remove"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "admin", model.ConvRoleAdmin)
	_ = convRepo.AddMember(ctx, convID, "target", model.ConvRoleMember)

	err := mgr.RemoveMember(ctx, convID, "target", "admin")
	if err != nil {
		t.Fatalf("RemoveMember by admin returned error: %v", err)
	}

	ok, err := mgr.IsMember(ctx, convID, "target")
	if err != nil {
		t.Fatalf("IsMember error: %v", err)
	}
	if ok {
		t.Errorf("expected target to be removed")
	}
}

func TestRemoveMember_SelfAlwaysWorks(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_self_leave"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "user1", model.ConvRoleMember)

	err := mgr.RemoveMember(ctx, convID, "user1", "user1")
	if err != nil {
		t.Fatalf("RemoveMember (self-leave) returned error: %v", err)
	}

	ok, err := mgr.IsMember(ctx, convID, "user1")
	if err != nil {
		t.Fatalf("IsMember error: %v", err)
	}
	if ok {
		t.Errorf("expected user1 to be removed after leave")
	}
}

func TestRemoveMember_NonAdminNoPermission(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_nonadmin_no_perm"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "member_a", model.ConvRoleMember)
	_ = convRepo.AddMember(ctx, convID, "member_b", model.ConvRoleMember)

	err := mgr.RemoveMember(ctx, convID, "member_b", "member_a")
	if err == nil {
		t.Fatal("expected error when non-admin removes another member")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNoPermission {
		t.Errorf("expected ErrNoPermission code, got %d", appErr.Code)
	}
}

func TestRemoveMember_NonExistentConv(t *testing.T) {
	mgr, _, _, _ := newManager()
	ctx := context.Background()

	err := mgr.RemoveMember(ctx, "ghost", "someone", "anyone")
	if err == nil {
		t.Fatal("expected error for non-existent conversation")
	}
	appErr, ok := err.(*model.AppError)
	if !ok {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNotFound {
		t.Errorf("expected ErrNotFound code, got %d", appErr.Code)
	}
}

func TestLeave_DelegatesToRemoveMember(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_leave"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "leaver", model.ConvRoleMember)

	err := mgr.Leave(ctx, convID, "leaver")
	if err != nil {
		t.Fatalf("Leave returned error: %v", err)
	}

	ok, err := mgr.IsMember(ctx, convID, "leaver")
	if err != nil {
		t.Fatalf("IsMember error: %v", err)
	}
	if ok {
		t.Errorf("expected leaver to no longer be a member")
	}
}

func TestIsMember_DelegatesToRepo(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_ismember"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "member", model.ConvRoleMember)

	ok, err := mgr.IsMember(ctx, convID, "member")
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if !ok {
		t.Errorf("expected member to be found")
	}

	ok, err = mgr.IsMember(ctx, convID, "stranger")
	if err != nil {
		t.Fatalf("IsMember returned error: %v", err)
	}
	if ok {
		t.Errorf("expected stranger not to be a member")
	}
}

func TestGetMembers_DelegatesToRepo(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_get_members"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "a", model.ConvRoleMember)
	_ = convRepo.AddMember(ctx, convID, "b", model.ConvRoleMember)

	members, err := mgr.GetMembers(ctx, convID)
	if err != nil {
		t.Fatalf("GetMembers returned error: %v", err)
	}
	if len(members) != 2 {
		t.Errorf("expected 2 members, got %d", len(members))
	}
}

func TestNewManager(t *testing.T) {
	convRepo := newMockConvRepo()
	msgRepo := &mockMsgRepo{}
	userRepo := newMockUserRepo()
	seqCache := newMockSeqCache()

	mgr := NewManager(convRepo, msgRepo, seqCache, userRepo, nil)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}

// ---------------------------------------------------------------------------
// Mock joinRequestRepo
// ---------------------------------------------------------------------------

type mockJoinRequestRepo struct {
	mu       sync.Mutex
	requests map[string]*model.JoinRequest // key: convID + ":" + userID
}

func newMockJoinRequestRepo() *mockJoinRequestRepo {
	return &mockJoinRequestRepo{requests: make(map[string]*model.JoinRequest)}
}

func (r *mockJoinRequestRepo) key(convID, userID string) string {
	return convID + ":" + userID
}

func (r *mockJoinRequestRepo) Create(_ context.Context, convID, userID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(convID, userID)
	r.requests[k] = &model.JoinRequest{ConvID: convID, UserID: userID, Status: model.JoinRequestPending}
	return nil
}

func (r *mockJoinRequestRepo) Get(_ context.Context, convID, userID string) (*model.JoinRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(convID, userID)
	jr, ok := r.requests[k]
	if !ok {
		return nil, nil
	}
	cp := *jr
	return &cp, nil
}

func (r *mockJoinRequestRepo) ListByConv(_ context.Context, convID string, status model.JoinRequestStatus) ([]*model.JoinRequest, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	var result []*model.JoinRequest
	for _, jr := range r.requests {
		if jr.ConvID == convID && jr.Status == status {
			cp := *jr
			result = append(result, &cp)
		}
	}
	return result, nil
}

func (r *mockJoinRequestRepo) UpdateStatus(_ context.Context, convID, userID string, status model.JoinRequestStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(convID, userID)
	jr, ok := r.requests[k]
	if !ok {
		return fmt.Errorf("join request not found")
	}
	jr.Status = status
	return nil
}

func (r *mockJoinRequestRepo) ExistsPending(_ context.Context, convID, userID string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	k := r.key(convID, userID)
	jr, ok := r.requests[k]
	return ok && jr.Status == model.JoinRequestPending, nil
}

// ---------------------------------------------------------------------------
// Join Request tests
// ---------------------------------------------------------------------------

func newManagerWithJR() (*Manager, *mockConvRepo, *mockUserRepo, *mockJoinRequestRepo) {
	convRepo := newMockConvRepo()
	msgRepo := &mockMsgRepo{}
	userRepo := newMockUserRepo()
	seqCache := newMockSeqCache()
	jrRepo := newMockJoinRequestRepo()
	return NewManager(convRepo, msgRepo, seqCache, userRepo, jrRepo), convRepo, userRepo, jrRepo
}

func TestRequestJoin_Success(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleOwner)

	err := mgr.RequestJoin(ctx, "g1", "bob")
	if err != nil {
		t.Fatalf("RequestJoin failed: %v", err)
	}
	// verify the join request was created (pending)
	jr, _ := jrRepo.Get(ctx, "g1", "bob")
	if jr == nil || jr.Status != model.JoinRequestPending {
		t.Error("join request should be pending")
	}
}

func TestRequestJoin_ConvNotFound(t *testing.T) {
	mgr, _, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")

	err := mgr.RequestJoin(ctx, "nonexistent", "alice")
	if err == nil {
		t.Fatal("expected error for nonexistent conversation")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.conv_not_found_mgr" {
		t.Errorf("expected conv_not_found_mgr key, got %s", appErr.Key)
	}
}

func TestRequestJoin_NotGroup(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "p1", Type: model.ConvP2P})
	_ = convRepo.AddMember(ctx, "p1", "alice", model.ConvRoleOwner)

	err := mgr.RequestJoin(ctx, "p1", "bob")
	if err == nil {
		t.Fatal("expected error for P2P conversation")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.group_only" {
		t.Errorf("expected group_only key, got %s", appErr.Key)
	}
}

func TestRequestJoin_UserNotFound(t *testing.T) {
	mgr, convRepo, _, _ := newManagerWithJR()
	ctx := context.Background()
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})

	err := mgr.RequestJoin(ctx, "g1", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent user")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.user_not_found" {
		t.Errorf("expected user_not_found key, got %s", appErr.Key)
	}
}

func TestRequestJoin_AlreadyMember(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleOwner)

	err := mgr.RequestJoin(ctx, "g1", "alice")
	if err != model.ErrAlreadyMember {
		t.Errorf("expected ErrAlreadyMember, got %v", err)
	}
}

func TestRequestJoin_DuplicateRequest(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleOwner)

	// first request should succeed
	if err := mgr.RequestJoin(ctx, "g1", "bob"); err != nil {
		t.Fatalf("first request failed: %v", err)
	}
	// duplicate should fail
	err := mgr.RequestJoin(ctx, "g1", "bob")
	if err != model.ErrDuplicateRequest {
		t.Errorf("expected ErrDuplicateRequest, got %v", err)
	}
}

func TestListJoinRequests_Success(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)

	_ = jrRepo.Create(ctx, "g1", "bob")

	requests, err := mgr.ListJoinRequests(ctx, "g1", "alice")
	if err != nil {
		t.Fatalf("ListJoinRequests failed: %v", err)
	}
	if len(requests) != 1 {
		t.Errorf("expected 1 request, got %d", len(requests))
	}
}

func TestListJoinRequests_PermissionDenied(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // member
	userRepo.addUser("admin") // admin
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "admin", model.ConvRoleAdmin)
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleMember)

	_, err := mgr.ListJoinRequests(ctx, "g1", "alice")
	if err == nil {
		t.Fatal("expected error for non-admin")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.permission_denied" {
		t.Errorf("expected permission_denied key, got %s", appErr.Key)
	}
}

func TestApproveJoinRequest_Success(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // admin
	userRepo.addUser("bob")   // requester
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)
	_ = jrRepo.Create(ctx, "g1", "bob")

	err := mgr.ApproveJoinRequest(ctx, "g1", "bob", "alice")
	if err != nil {
		t.Fatalf("ApproveJoinRequest failed: %v", err)
	}
	// verify bob is now a member
	isMember, _ := convRepo.IsMember(ctx, "g1", "bob")
	if !isMember {
		t.Error("bob should be a member after approval")
	}
	// verify request status updated
	jr, _ := jrRepo.Get(ctx, "g1", "bob")
	if jr == nil || jr.Status != model.JoinRequestApproved {
		t.Error("request should be approved")
	}
}

func TestApproveJoinRequest_NoPendingRequest(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)

	err := mgr.ApproveJoinRequest(ctx, "g1", "bob", "alice")
	if err != model.ErrNoPendingRequest {
		t.Errorf("expected ErrNoPendingRequest, got %v", err)
	}
}

func TestApproveJoinRequest_PermissionDenied(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // member
	userRepo.addUser("bob")   // requester
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleMember)
	_ = jrRepo.Create(ctx, "g1", "bob")

	err := mgr.ApproveJoinRequest(ctx, "g1", "bob", "alice")
	if err == nil {
		t.Fatal("expected error for non-admin")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.permission_denied" {
		t.Errorf("expected permission_denied key, got %s", appErr.Key)
	}
}

func TestApproveJoinRequest_GroupFull(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // admin
	userRepo.addUser("bob")   // requester
	// Create a group with max 1 member (admin already takes one slot)
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup, MaxMembers: 1})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)
	_ = jrRepo.Create(ctx, "g1", "bob")

	err := mgr.ApproveJoinRequest(ctx, "g1", "bob", "alice")
	if err == nil {
		t.Fatal("expected error for full group")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.group_full" {
		t.Errorf("expected group_full key, got %s", appErr.Key)
	}
}

func TestRejectJoinRequest_Success(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // admin
	userRepo.addUser("bob")   // requester
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)
	_ = jrRepo.Create(ctx, "g1", "bob")

	err := mgr.RejectJoinRequest(ctx, "g1", "bob", "alice")
	if err != nil {
		t.Fatalf("RejectJoinRequest failed: %v", err)
	}
	jr, _ := jrRepo.Get(ctx, "g1", "bob")
	if jr == nil || jr.Status != model.JoinRequestRejected {
		t.Error("request should be rejected")
	}
}

func TestRejectJoinRequest_NoPendingRequest(t *testing.T) {
	mgr, convRepo, userRepo, _ := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice")
	userRepo.addUser("bob")
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleAdmin)

	err := mgr.RejectJoinRequest(ctx, "g1", "bob", "alice")
	if err != model.ErrNoPendingRequest {
		t.Errorf("expected ErrNoPendingRequest, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// GetOrCreateSystemConv tests
// ---------------------------------------------------------------------------

func TestGetOrCreateSystemConv_CreatesNew(t *testing.T) {
	mgr, convRepo, _, seqCache := newManager()
	ctx := context.Background()
	userID := "user_sys"

	conv, err := mgr.GetOrCreateSystemConv(ctx, userID)
	if err != nil {
		t.Fatalf("GetOrCreateSystemConv failed: %v", err)
	}
	expectedID := model.MakeSystemConvID(userID)
	if conv.ConvID != expectedID {
		t.Errorf("convID = %q, want %q", conv.ConvID, expectedID)
	}
	if conv.Type != model.ConvSystem {
		t.Errorf("type = %v, want ConvSystem", conv.Type)
	}
	if conv.OwnerID != userID {
		t.Errorf("ownerID = %q, want %q", conv.OwnerID, userID)
	}

	isMember, err := convRepo.IsMember(ctx, expectedID, userID)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("user should be a member of system conversation")
	}

	role, err := convRepo.GetMemberRole(ctx, expectedID, userID)
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != model.ConvRoleOwner {
		t.Errorf("role = %v, want Owner", role)
	}

	seqCache.mu.Lock()
	_, ok := seqCache.seqs[expectedID]
	seqCache.mu.Unlock()
	if !ok {
		t.Error("expected conv seq to be initialized")
	}
}

func TestGetOrCreateSystemConv_ReturnsExisting(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()
	userID := "user_existing"

	// Create once
	conv1, err := mgr.GetOrCreateSystemConv(ctx, userID)
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Get again
	conv2, err := mgr.GetOrCreateSystemConv(ctx, userID)
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	if conv1.ConvID != conv2.ConvID {
		t.Errorf("conversation IDs differ: %q vs %q", conv1.ConvID, conv2.ConvID)
	}

	// Ensure member still exists
	isMember, err := convRepo.IsMember(ctx, conv2.ConvID, userID)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("user should still be a member")
	}
}

func TestGetOrCreateSystemTx_CreatesNew(t *testing.T) {
	mgr, convRepo, _, seqCache := newManager()
	ctx := context.Background()
	userID := "user_tx"

	conv, err := mgr.GetOrCreateSystemConvTx(ctx, nil, userID)
	if err != nil {
		t.Fatalf("GetOrCreateSystemConvTx failed: %v", err)
	}
	expectedID := model.MakeSystemConvID(userID)
	if conv.ConvID != expectedID {
		t.Errorf("convID = %q, want %q", conv.ConvID, expectedID)
	}

	isMember, err := convRepo.IsMember(ctx, expectedID, userID)
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("user should be a member after tx creation")
	}

	seqCache.mu.Lock()
	_, ok := seqCache.seqs[expectedID]
	seqCache.mu.Unlock()
	if !ok {
		t.Error("expected conv seq to be initialized")
	}
}

func TestGetOrCreateSystemTx_ReturnsExisting(t *testing.T) {
	mgr, _, _, _ := newManager()
	ctx := context.Background()
	userID := "user_tx_existing"

	// Create first via non-tx
	conv1, err := mgr.GetOrCreateSystemConv(ctx, userID)
	if err != nil {
		t.Fatalf("first call (non-tx) failed: %v", err)
	}

	// Get via tx should return existing
	conv2, err := mgr.GetOrCreateSystemConvTx(ctx, nil, userID)
	if err != nil {
		t.Fatalf("second call (tx) failed: %v", err)
	}

	if conv1.ConvID != conv2.ConvID {
		t.Errorf("conversation IDs differ: %q vs %q", conv1.ConvID, conv2.ConvID)
	}
}

func TestGetMemberRole_ReturnsRole(t *testing.T) {
	mgr, convRepo, _, _ := newManager()
	ctx := context.Background()

	convID := "conv_get_role"
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: convID, Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, convID, "owner", model.ConvRoleOwner)
	_ = convRepo.AddMember(ctx, convID, "admin", model.ConvRoleAdmin)
	_ = convRepo.AddMember(ctx, convID, "member", model.ConvRoleMember)

	tests := []struct {
		userID   string
		expected model.ConvRole
	}{
		{"owner", model.ConvRoleOwner},
		{"admin", model.ConvRoleAdmin},
		{"member", model.ConvRoleMember},
	}

	for _, tt := range tests {
		t.Run(tt.userID, func(t *testing.T) {
			role, err := mgr.GetMemberRole(ctx, convID, tt.userID)
			if err != nil {
				t.Fatalf("GetMemberRole(%q): %v", tt.userID, err)
			}
			if role != tt.expected {
				t.Errorf("role = %v, want %v", role, tt.expected)
			}
		})
	}
}

func TestGetMemberRole_NonExistentConv(t *testing.T) {
	mgr, _, _, _ := newManager()
	ctx := context.Background()

	_, err := mgr.GetMemberRole(ctx, "nonexistent", "user")
	if err == nil {
		t.Fatal("expected error for non-existent conversation")
	}
}

func TestRejectJoinRequest_PermissionDenied(t *testing.T) {
	mgr, convRepo, userRepo, jrRepo := newManagerWithJR()
	ctx := context.Background()
	userRepo.addUser("alice") // member
	userRepo.addUser("bob")   // requester
	_ = convRepo.Create(ctx, &model.Conversation{ConvID: "g1", Type: model.ConvGroup})
	_ = convRepo.AddMember(ctx, "g1", "alice", model.ConvRoleMember)
	_ = jrRepo.Create(ctx, "g1", "bob")

	err := mgr.RejectJoinRequest(ctx, "g1", "bob", "alice")
	if err == nil {
		t.Fatal("expected error for non-admin")
	}
	if appErr, ok := err.(*model.AppError); ok && appErr.Key != "err.permission_denied" {
		t.Errorf("expected permission_denied key, got %s", appErr.Key)
	}
}
