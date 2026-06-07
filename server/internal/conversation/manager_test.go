package conversation

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/dolphinz/im-server/pkg/model"
)

// ---------------------------------------------------------------------------
// Mock convRepo
// ---------------------------------------------------------------------------

type mockConvRepo struct {
	mu       sync.Mutex
	convs    map[string]*model.Conversation
	members  map[string]map[string]*model.ConvMember // convID -> userID -> member
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
	mu    sync.Mutex
	seqs  map[string]int64
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

func newManager() (*Manager, *mockConvRepo, *mockSeqCache) {
	convRepo := newMockConvRepo()
	msgRepo := &mockMsgRepo{}
	seqCache := newMockSeqCache()
	return NewManager(convRepo, msgRepo, seqCache), convRepo, seqCache
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
	mgr, convRepo, _ := newManager()
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
	mgr, _, seqCache := newManager()
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
	mgr, _, seqCache := newManager()
	ctx := context.Background()

	ownerID := "owner"
	memberIDs := []string{"m1", "m2", "m3"}
	name := "test group"
	gen := counterIDGen()

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
	mgr, convRepo, _ := newManager()
	ctx := context.Background()

	ownerID := "owner"
	memberIDs := []string{"m1", "owner", "m2"} // owner appears in member list
	gen := counterIDGen()

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
	mgr, convRepo, _ := newManager()
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
	mgr, _, _ := newManager()
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
	mgr, convRepo, _ := newManager()
	ctx := context.Background()

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
	mgr, convRepo, _ := newManager()
	ctx := context.Background()

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
	mgr, _, _ := newManager()
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

func TestRemoveMember_ByAdminSucceeds(t *testing.T) {
	mgr, convRepo, _ := newManager()
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
	mgr, convRepo, _ := newManager()
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
	mgr, convRepo, _ := newManager()
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
	mgr, _, _ := newManager()
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
	mgr, convRepo, _ := newManager()
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
	mgr, convRepo, _ := newManager()
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
	mgr, convRepo, _ := newManager()
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
	seqCache := newMockSeqCache()

	mgr := NewManager(convRepo, msgRepo, seqCache)
	if mgr == nil {
		t.Fatal("NewManager returned nil")
	}
}
