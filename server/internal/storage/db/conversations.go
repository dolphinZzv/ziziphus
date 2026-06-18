package db

import (
	"context"
	"strings"
	"time"

	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type ConvRepo struct {
	pool DBPool
}

func NewConvRepo(pool DBPool) *ConvRepo {
	return &ConvRepo{pool: pool}
}

func (r *ConvRepo) Create(ctx context.Context, c *model.Conversation) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO conversations (conv_id, type, name, owner_id, avatar, max_members, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		c.ConvID, c.Type, c.Name, c.OwnerID, c.Avatar, c.MaxMembers, time.UnixMilli(c.CreatedAt))
	return err
}

func (r *ConvRepo) Get(ctx context.Context, convID string) (*model.Conversation, error) {
	c := &model.Conversation{}
	var lastMsgAt *time.Time
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT conv_id, type, name, owner_id, avatar, max_members, last_msg_id, last_msg_at, created_at
		 FROM conversations WHERE conv_id = $1`, convID).
		Scan(&c.ConvID, &c.Type, &c.Name, &c.OwnerID, &c.Avatar, &c.MaxMembers, &c.LastMsgID, &lastMsgAt, &createdAt)
	if err != nil {
		return nil, err
	}
	c.CreatedAt = createdAt.UnixMilli()
	if lastMsgAt != nil {
		c.LastMsgAt = lastMsgAt.UnixMilli()
	}
	return c, nil
}

func (r *ConvRepo) UpdateLastMsg(ctx context.Context, convID string, msgID int64) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET last_msg_id = $1, last_msg_at = NOW() WHERE conv_id = $2`, msgID, convID)
	return err
}

type ConvListItem struct {
	ConvID      string           `json:"conv_id"`
	Type        model.ConvType   `json:"type"`
	Name        string           `json:"name"`
	Avatar      string           `json:"avatar"`
	UnreadCount int64            `json:"unread_count"`
	LastMessage *LastMessageInfo `json:"last_message,omitempty"`
	LastMsgAt   int64            `json:"last_msg_at"`
	Role        model.ConvRole   `json:"role"`
	Mute        bool             `json:"mute"`
	MentionMe   bool             `json:"mention_me"`
	PartnerType int              `json:"partner_type"`
}

type LastMessageInfo struct {
	MsgID       int64  `json:"msg_id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	Body        string `json:"body"`
	ContentType int    `json:"content_type"`
	Timestamp   int64  `json:"timestamp"`
	Status      int    `json:"status"`
}

func (r *ConvRepo) GetUserConvs(ctx context.Context, userID string, page, size int) ([]*ConvListItem, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conv_members WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT c.conv_id, c.type, c.name, c.avatar,
		        COALESCE(m.msg_id, 0), COALESCE(m.sender_id, ''), COALESCE(u.name, ''), COALESCE(m.body, ''), COALESCE(m.content_type, 0), COALESCE(m.timestamp, 0), COALESCE(m.status, 0),
		        c.last_msg_at,
		        cm.role, cm.mute
		 FROM conv_members cm
		 JOIN conversations c ON c.conv_id = cm.conv_id
			 LEFT JOIN messages m ON m.msg_id = c.last_msg_id
			 LEFT JOIN users u ON u.id = m.sender_id
		 WHERE cm.user_id = $1
		 ORDER BY c.last_msg_at DESC NULLS LAST
		 LIMIT $2 OFFSET $3`, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]*ConvListItem, 0)
	for rows.Next() {
		item := &ConvListItem{}
		var lastMsgID int64
		var lastMsgSenderID string
		var lastMsgSenderName string
		var lastMsgBody string
		var lastMsgContentType int
		var lastMsgTimestamp int64
		var lastMsgStatus int
		var lastMsgAt *time.Time
		if err := rows.Scan(&item.ConvID, &item.Type, &item.Name, &item.Avatar,
			&lastMsgID, &lastMsgSenderID, &lastMsgSenderName, &lastMsgBody, &lastMsgContentType, &lastMsgTimestamp, &lastMsgStatus,
			&lastMsgAt, &item.Role, &item.Mute); err != nil {
			return nil, 0, err
		}
		if lastMsgID > 0 {
			item.LastMessage = &LastMessageInfo{
				MsgID:       lastMsgID,
				SenderID:    lastMsgSenderID,
				SenderName:  lastMsgSenderName,
				Body:        lastMsgBody,
				ContentType: lastMsgContentType,
				Timestamp:   lastMsgTimestamp,
				Status:      lastMsgStatus,
			}
		}
		if lastMsgAt != nil {
			item.LastMsgAt = lastMsgAt.UnixMilli()
		}
		items = append(items, item)
	}

	// For P2P conversations, resolve the partner's display name from users table
	partnerIDs := make([]string, 0, len(items))
	itemIndexByPartner := make(map[string][]int) // partnerID -> item indices
	for i, item := range items {
		if item.Type == model.ConvP2P {
			parts := strings.Split(item.ConvID, ":")
			if len(parts) == 2 {
				partnerID := parts[0]
				if partnerID == userID {
					partnerID = parts[1]
				}
				item.Name = partnerID // fallback to partner ID
				partnerIDs = append(partnerIDs, partnerID)
				itemIndexByPartner[partnerID] = append(itemIndexByPartner[partnerID], i)
			}
		}
	}
	if len(partnerIDs) > 0 {
		// Resolve partner names with nickname priority (contacts.nickname > users.name)
		type partnerInfo struct {
			id, name, nickname string
			userType           int
		}
		partnerMap := make(map[string]*partnerInfo, len(partnerIDs))
		rows, err := r.pool.Query(ctx,
			`SELECT u.id, u.name, u.type, COALESCE(c.nickname, '')
			 FROM users u
			 LEFT JOIN contacts c ON c.user_id = $1 AND c.contact_id = u.id
			 WHERE u.id = ANY($2)`, userID, partnerIDs)
		if err != nil {
			logger.Error("resolve p2p names query failed", "error", err)
		} else {
			defer rows.Close()
			for rows.Next() {
				var info partnerInfo
				if err := rows.Scan(&info.id, &info.name, &info.userType, &info.nickname); err != nil {
					continue
				}
				partnerMap[info.id] = &info
			}
			for partnerID, indices := range itemIndexByPartner {
				if info, ok := partnerMap[partnerID]; ok {
					displayName := info.name
					if info.nickname != "" {
						displayName = info.nickname
					}
					for _, idx := range indices {
						items[idx].Name = displayName
						items[idx].PartnerType = info.userType
					}
				}
			}
		}
	}

	return items, total, nil
}

func (r *ConvRepo) AddMember(ctx context.Context, convID, userID string, role model.ConvRole) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO conv_members (conv_id, user_id, role, joined_at)
		 VALUES ($1, $2, $3, NOW()) ON CONFLICT (conv_id, user_id) DO NOTHING`,
		convID, userID, role)
	return err
}

func (r *ConvRepo) RemoveMember(ctx context.Context, convID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM conv_members WHERE conv_id = $1 AND user_id = $2`, convID, userID)
	return err
}

func (r *ConvRepo) GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT cm.conv_id, cm.user_id, cm.role, cm.nickname, cm.mute, cm.joined_at,
		        COALESCE(u.type, 0), COALESCE(u.wake_mode, 0)
		 FROM conv_members cm
		 LEFT JOIN users u ON u.id = cm.user_id
		 WHERE cm.conv_id = $1 ORDER BY cm.joined_at`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []*model.ConvMember
	for rows.Next() {
		m := &model.ConvMember{}
		var joinedAt time.Time
		if err := rows.Scan(&m.ConvID, &m.UserID, &m.Role, &m.Nickname, &m.Mute, &joinedAt,
			&m.UserType, &m.WakeMode); err != nil {
			return nil, err
		}
		m.JoinedAt = joinedAt.UnixMilli()
		members = append(members, m)
	}
	return members, nil
}

func (r *ConvRepo) IsMember(ctx context.Context, convID, userID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conv_members WHERE conv_id = $1 AND user_id = $2`, convID, userID).Scan(&count)
	return count > 0, err
}

func (r *ConvRepo) GetMemberRole(ctx context.Context, convID, userID string) (model.ConvRole, error) {
	var role model.ConvRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM conv_members WHERE conv_id = $1 AND user_id = $2`, convID, userID).Scan(&role)
	return role, err
}

type GroupSearchItem struct {
	ConvID      string `json:"conv_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar"`
	OwnerID     string `json:"owner_id"`
	MemberCount int    `json:"member_count"`
	CreatedAt   int64  `json:"created_at"`
}

func (r *ConvRepo) SearchByName(ctx context.Context, q string, page, size int) ([]*GroupSearchItem, int, error) {
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM conversations WHERE type = $1 AND name ILIKE $2`,
		model.ConvGroup, "%"+q+"%").Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT c.conv_id, c.name, c.avatar, c.owner_id, COALESCE(mc.count, 0), c.created_at
		 FROM conversations c
		 LEFT JOIN (SELECT conv_id, COUNT(*) AS count FROM conv_members GROUP BY conv_id) mc ON mc.conv_id = c.conv_id
		 WHERE c.type = $1 AND c.name ILIKE $2
		 ORDER BY c.created_at DESC
		 LIMIT $3 OFFSET $4`,
		model.ConvGroup, "%"+q+"%", size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []*GroupSearchItem
	for rows.Next() {
		item := &GroupSearchItem{}
		var createdAt time.Time
		if err := rows.Scan(&item.ConvID, &item.Name, &item.Avatar, &item.OwnerID, &item.MemberCount, &createdAt); err != nil {
			return nil, 0, err
		}
		item.CreatedAt = createdAt.UnixMilli()
		items = append(items, item)
	}
	return items, total, nil
}

func (r *ConvRepo) UpdateNameAvatar(ctx context.Context, convID, name, avatar string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET name = $1, avatar = $2 WHERE conv_id = $3`, name, avatar, convID)
	return err
}
