package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type ConvRepo struct {
	pool *pgxpool.Pool
}

func NewConvRepo(pool *pgxpool.Pool) *ConvRepo {
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
	MentionMe   bool             `json:"mention_me"`
}

type LastMessageInfo struct {
	MsgID       int64  `json:"msg_id"`
	SenderID    string `json:"sender_id"`
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
		        COALESCE(m.msg_id, 0), COALESCE(m.sender_id, ''), COALESCE(m.body, ''), COALESCE(m.content_type, 0), COALESCE(m.timestamp, 0), COALESCE(m.status, 0),
		        c.last_msg_at,
		        cm.role, cm.mute
		 FROM conv_members cm
		 JOIN conversations c ON c.conv_id = cm.conv_id
		 LEFT JOIN messages m ON m.msg_id = c.last_msg_id
		 WHERE cm.user_id = $1
		 ORDER BY c.last_msg_at DESC NULLS LAST
		 LIMIT $2 OFFSET $3`, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []*ConvListItem
	for rows.Next() {
		item := &ConvListItem{}
		var lastMsgID int64
		var lastMsgSenderID string
		var lastMsgBody string
		var lastMsgContentType int
		var lastMsgTimestamp int64
		var lastMsgStatus int
		var lastMsgAt *time.Time
		var role model.ConvRole
		var mute bool
		if err := rows.Scan(&item.ConvID, &item.Type, &item.Name, &item.Avatar,
			&lastMsgID, &lastMsgSenderID, &lastMsgBody, &lastMsgContentType, &lastMsgTimestamp, &lastMsgStatus,
			&lastMsgAt, &role, &mute); err != nil {
			return nil, 0, err
		}
		if lastMsgID > 0 {
			item.LastMessage = &LastMessageInfo{
				MsgID:       lastMsgID,
				SenderID:    lastMsgSenderID,
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
		`SELECT conv_id, user_id, role, nickname, mute, joined_at
		 FROM conv_members WHERE conv_id = $1 ORDER BY joined_at`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []*model.ConvMember
	for rows.Next() {
		m := &model.ConvMember{}
		var joinedAt time.Time
		if err := rows.Scan(&m.ConvID, &m.UserID, &m.Role, &m.Nickname, &m.Mute, &joinedAt); err != nil {
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

func (r *ConvRepo) UpdateNameAvatar(ctx context.Context, convID, name, avatar string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conversations SET name = $1, avatar = $2 WHERE conv_id = $3`, name, avatar, convID)
	return err
}
