package conversation

import (
	"context"
	"time"

	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
)

type Manager struct {
	convRepo convRepo
	msgRepo  msgRepo
	seqCache seqCache
}

type convRepo interface {
	Create(ctx context.Context, c *model.Conversation) error
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	UpdateLastMsg(ctx context.Context, convID string, msgID int64) error
	AddMember(ctx context.Context, convID, userID string, role model.ConvRole) error
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

func NewManager(convRepo convRepo, msgRepo msgRepo, seqCache seqCache) *Manager {
	return &Manager{
		convRepo: convRepo,
		msgRepo:  msgRepo,
		seqCache: seqCache,
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

func (m *Manager) CreateGroup(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error) {
	convID := model.GenerateGroupConvID(idGen)
	now := time.Now().UnixMilli()
	conv := &model.Conversation{
		ConvID:    convID,
		Type:      model.ConvGroup,
		Name:      name,
		OwnerID:   ownerID,
		MaxMembers: 200,
		CreatedAt: now,
	}
	if err := m.convRepo.Create(ctx, conv); err != nil {
		return nil, err
	}

	// add owner
	m.convRepo.AddMember(ctx, convID, ownerID, model.ConvRoleOwner)
	// add members
	for _, mid := range memberIDs {
		if mid != ownerID {
			m.convRepo.AddMember(ctx, convID, mid, model.ConvRoleMember)
		}
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
	// verify operator is member
	role, err := m.convRepo.GetMemberRole(ctx, convID, operatorID)
	if err != nil {
		return &model.AppError{Code: model.ErrNotFound, Message: "会话不存在", Key: "err.conv_not_found_mgr"}
	}
	if role < model.ConvRoleAdmin {
		return &model.AppError{Code: model.ErrNoPermission, Message: "权限不足", Key: "err.permission_denied"}
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
