package conversation

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type Manager struct {
	convRepo        convRepo
	msgRepo         msgRepo
	seqCache        seqCache
	userRepo        userRepo
	joinRequestRepo joinRequestRepo
}

type userRepo interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
}

type convRepo interface {
	IsDirectChatBlocked(ctx context.Context, userID string) (bool, error)
	Create(ctx context.Context, c *model.Conversation) error
	CreateTx(ctx context.Context, tx pgx.Tx, c *model.Conversation) error
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	UpdateLastMsg(ctx context.Context, convID string, msgID int64) error
	AddMember(ctx context.Context, convID, userID string, role model.ConvRole) error
	AddMemberTx(ctx context.Context, tx pgx.Tx, convID, userID string, role model.ConvRole) error
	RemoveMember(ctx context.Context, convID, userID string) error
	GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
	GetMemberRole(ctx context.Context, convID, userID string) (model.ConvRole, error)
}

type msgRepo interface {
	GetMaxConvSeq(ctx context.Context, convID string) (int64, error)
}

type seqCache interface {
	InitConvSeq(ctx context.Context, convID string, seq int64) error
}

type joinRequestRepo interface {
	Create(ctx context.Context, convID, userID string) error
	Get(ctx context.Context, convID, userID string) (*model.JoinRequest, error)
	ListByConv(ctx context.Context, convID string, status model.JoinRequestStatus) ([]*model.JoinRequest, error)
	UpdateStatus(ctx context.Context, convID, userID string, status model.JoinRequestStatus) error
	ExistsPending(ctx context.Context, convID, userID string) (bool, error)
}

func NewManager(convRepo convRepo, msgRepo msgRepo, seqCache seqCache, userRepo userRepo, joinRequestRepo joinRequestRepo) *Manager {
	return &Manager{
		convRepo:        convRepo,
		msgRepo:         msgRepo,
		seqCache:        seqCache,
		userRepo:        userRepo,
		joinRequestRepo: joinRequestRepo,
	}
}

func (m *Manager) GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error) {
	convID := model.MakeP2PConvID(userA, userB)
	conv, err := m.convRepo.Get(ctx, convID)
	if err == nil {
		return conv, nil
	}

	// create conversation + members atomically
	now := time.Now().UnixMilli()
	conv = &model.Conversation{
		ConvID:    convID,
		Type:      model.ConvP2P,
		Name:      "",
		OwnerID:   userA,
		CreatedAt: now,
	}
	if err := m.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	m.convRepo.AddMember(ctx, convID, userA, model.ConvRoleOwner)
	m.convRepo.AddMember(ctx, convID, userB, model.ConvRoleMember)

	// init conv seq to 0
	m.seqCache.InitConvSeq(ctx, convID, 0)
	logger.Info("P2P conversation created", "conv_id", convID)
	return conv, nil
}

// GetOrCreateSystemConv returns the system conversation for the given user.
// If it does not exist it is lazily created with the user as the sole owner member.
func (m *Manager) GetOrCreateSystemConv(ctx context.Context, userID string) (*model.Conversation, error) {
	convID := model.MakeSystemConvID(userID)
	conv, err := m.convRepo.Get(ctx, convID)
	if err == nil {
		// Ensure the user is a member (repair if creation previously failed).
		if isMember, _ := m.convRepo.IsMember(ctx, convID, userID); !isMember {
			if err := m.convRepo.AddMember(ctx, convID, userID, model.ConvRoleOwner); err != nil {
				logger.Error("repair system conv member failed", "conv_id", convID, "user_id", userID, "error", err)
			}
		}
		return conv, nil
	}

	now := time.Now().UnixMilli()
	conv = &model.Conversation{
		ConvID:    convID,
		Type:      model.ConvSystem,
		Name:      "",
		OwnerID:   userID,
		CreatedAt: now,
	}
	if err := m.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}
	if err := m.convRepo.AddMember(ctx, convID, userID, model.ConvRoleOwner); err != nil {
		return nil, fmt.Errorf("add system conv member: %w", err)
	}
	m.seqCache.InitConvSeq(ctx, convID, 0)
	logger.Info("system conversation created", "conv_id", convID)
	return conv, nil
}

// GetOrCreateSystemConvTx is the transactional variant of GetOrCreateSystemConv.
func (m *Manager) GetOrCreateSystemConvTx(ctx context.Context, tx pgx.Tx, userID string) (*model.Conversation, error) {
	convID := model.MakeSystemConvID(userID)
	conv, err := m.convRepo.Get(ctx, convID)
	if err == nil {
		if isMember, _ := m.convRepo.IsMember(ctx, convID, userID); !isMember {
			m.convRepo.AddMemberTx(ctx, tx, convID, userID, model.ConvRoleOwner)
		}
		return conv, nil
	}

	now := time.Now().UnixMilli()
	conv = &model.Conversation{
		ConvID:    convID,
		Type:      model.ConvSystem,
		Name:      "",
		OwnerID:   userID,
		CreatedAt: now,
	}
	if err := m.convRepo.CreateTx(ctx, tx, conv); err != nil {
		return nil, err
	}
	if err := m.convRepo.AddMemberTx(ctx, tx, convID, userID, model.ConvRoleOwner); err != nil {
		return nil, fmt.Errorf("add system conv member (tx): %w", err)
	}
	m.seqCache.InitConvSeq(ctx, convID, 0)
	logger.Info("system conversation created (tx)", "conv_id", convID)
	return conv, nil
}

func (m *Manager) CreateGroup(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
	// deduplicate and remove owner from member list
	seen := map[string]struct{}{ownerID: {}}
	var uniqueMembers []string
	for _, mid := range memberIDs {
		if _, ok := seen[mid]; !ok {
			seen[mid] = struct{}{}
			uniqueMembers = append(uniqueMembers, mid)
		}
	}
	// 1. Check all members exist
	users, err := m.userRepo.GetByIDs(ctx, uniqueMembers)
	if err != nil {
		return nil, err
	}
	for _, mid := range uniqueMembers {
		if _, ok := users[mid]; !ok {
			return nil, &model.AppError{Code: model.ErrNotFound, Message: fmt.Sprintf("用户 %s 不存在", mid), Key: "err.user_not_found"}
		}
	}

	convID := model.GenerateGroupConvID(idGen)
	now := time.Now().UnixMilli()
	maxMembers := 200
	if len(uniqueMembers)+1 > maxMembers {
		return nil, &model.AppError{Code: model.ErrTooLarge, Message: "群组人数已达上限", Key: "err.group_full"}
	}
	conv := &model.Conversation{
		ConvID:     convID,
		Type:       model.ConvGroup,
		Name:       name,
		OwnerID:    ownerID,
		MaxMembers: maxMembers,
		CreatedAt:  now,
	}
	if err := m.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	// add owner
	m.convRepo.AddMember(ctx, convID, ownerID, model.ConvRoleOwner)
	// add members
	for _, mid := range uniqueMembers {
		m.convRepo.AddMember(ctx, convID, mid, model.ConvRoleMember)
	}
	// init conv seq
	m.seqCache.InitConvSeq(ctx, convID, 0)
	logger.Info("group created", "conv_id", convID, "name", name)
	return conv, nil
}

func (m *Manager) Get(ctx context.Context, convID string) (*model.Conversation, error) {
	return m.convRepo.Get(ctx, convID)
}

func (m *Manager) AddMember(ctx context.Context, convID, userID string, operatorID string) error {
	// 2. Check conversation exists and is a group
	conv, err := m.convRepo.Get(ctx, convID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if conv.Type != model.ConvGroup {
		return &model.AppError{Code: model.ErrBadMessage, Message: "仅支持群组操作", Key: "err.group_only"}
	}
	// 1. Check target user exists
	if _, err := m.userRepo.GetByID(ctx, userID); err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "用户不存在", Key: "err.user_not_found"}
	}
	// Verify operator is member and has admin/owner role
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
	}
	// 3. Check max members limit
	members, err := m.convRepo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}
	if conv.MaxMembers > 0 && len(members) >= conv.MaxMembers {
		return &model.AppError{Code: model.ErrTooLarge, Message: "群组人数已达上限", Key: "err.group_full"}
	}
	return m.convRepo.AddMember(ctx, convID, userID, model.ConvRoleMember)
}

func (m *Manager) RemoveMember(ctx context.Context, convID, userID, operatorID string) error {
	if operatorID == userID {
		// leaving self
		return m.convRepo.RemoveMember(ctx, convID, userID)
	}
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
	}
	return m.convRepo.RemoveMember(ctx, convID, userID)
}

func (m *Manager) Leave(ctx context.Context, convID, userID string) error {
	return m.convRepo.RemoveMember(ctx, convID, userID)
}

func (m *Manager) GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error) {
	return m.convRepo.GetMembers(ctx, convID)
}

func (m *Manager) IsMember(ctx context.Context, convID, userID string) (bool, error) {
	return m.convRepo.IsMember(ctx, convID, userID)
}

func (m *Manager) IsDirectChatBlocked(ctx context.Context, userID string) (bool, error) {
	return m.convRepo.IsDirectChatBlocked(ctx, userID)
}

func (m *Manager) GetMemberRole(ctx context.Context, convID, userID string) (model.ConvRole, error) {
	return m.convRepo.GetMemberRole(ctx, convID, userID)
}

func (m *Manager) RequestJoin(ctx context.Context, convID, userID string) error {
	conv, err := m.convRepo.Get(ctx, convID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if conv.Type != model.ConvGroup {
		return &model.AppError{Code: model.ErrBadMessage, Message: "仅支持群组", Key: "err.group_only"}
	}
	if _, err := m.userRepo.GetByID(ctx, userID); err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "用户不存在", Key: "err.user_not_found"}
	}
	isMember, err := m.convRepo.IsMember(ctx, convID, userID)
	if err != nil {
		return err
	}
	if isMember {
		return model.ErrAlreadyMember
	}
	exists, err := m.joinRequestRepo.ExistsPending(ctx, convID, userID)
	if err != nil {
		return err
	}
	if exists {
		return model.ErrDuplicateRequest
	}
	return m.joinRequestRepo.Create(ctx, convID, userID)
}

func (m *Manager) ListJoinRequests(ctx context.Context, convID, operatorID string) ([]*model.JoinRequest, error) {
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return nil, &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return nil, &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
	}
	return m.joinRequestRepo.ListByConv(ctx, convID, model.JoinRequestPending)
}

func (m *Manager) ApproveJoinRequest(ctx context.Context, convID, userID, operatorID string) error {
	jr, err := m.joinRequestRepo.Get(ctx, convID, userID)
	if err != nil {
		return err
	}
	if jr == nil || jr.Status != model.JoinRequestPending {
		return model.ErrNoPendingRequest
	}
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
	}
	conv, err := m.convRepo.Get(ctx, convID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	members, err := m.convRepo.GetMembers(ctx, convID)
	if err != nil {
		return err
	}
	if conv.MaxMembers > 0 && len(members) >= conv.MaxMembers {
		return &model.AppError{Code: model.ErrTooLarge, Message: "群组人数已达上限", Key: "err.group_full"}
	}
	if err := m.convRepo.AddMember(ctx, convID, userID, model.ConvRoleMember); err != nil {
		return err
	}
	return m.joinRequestRepo.UpdateStatus(ctx, convID, userID, model.JoinRequestApproved)
}

func (m *Manager) RejectJoinRequest(ctx context.Context, convID, userID, operatorID string) error {
	jr, err := m.joinRequestRepo.Get(ctx, convID, userID)
	if err != nil {
		return err
	}
	if jr == nil || jr.Status != model.JoinRequestPending {
		return model.ErrNoPendingRequest
	}
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
	}
	return m.joinRequestRepo.UpdateStatus(ctx, convID, userID, model.JoinRequestRejected)
}
