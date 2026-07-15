package model

type ConvType int

const (
	ConvP2P    ConvType = 1
	ConvGroup  ConvType = 2
	ConvSystem ConvType = 3
)

type ConvRole int

const (
	ConvRoleMember ConvRole = 0
	ConvRoleAdmin  ConvRole = 1
	ConvRoleOwner  ConvRole = 2
)

type Conversation struct {
	ConvID     string         `json:"conv_id"`
	Type       ConvType       `json:"type"`
	Name       string         `json:"name"`
	OwnerID    string         `json:"owner_id"`
	Avatar     string         `json:"avatar,omitempty"`
	Cover      string         `json:"cover,omitempty"`
	Notice     string         `json:"notice,omitempty"`
	Headline   string         `json:"headline,omitempty"`
	MaxMembers int            `json:"max_members,omitempty"`
	LastMsgID  int64          `json:"last_msg_id,omitempty"`
	LastMsgAt  int64          `json:"last_msg_at,omitempty"`
	CreatedAt  int64          `json:"created_at"`
	Settings   map[string]any `json:"settings,omitempty"`
}

type ConvMember struct {
	ConvID   string   `json:"conv_id"`
	UserID   string   `json:"user_id"`
	Role     ConvRole `json:"role"`
	Nickname string   `json:"nickname,omitempty"`
	Mute     bool     `json:"mute"`
	JoinedAt int64    `json:"joined_at"`
	UserType UserType `json:"user_type"`
	WakeMode WakeMode `json:"wake_mode"`
	Pinned   bool     `json:"pinned"`
}

type JoinRequestStatus int

const (
	JoinRequestPending  JoinRequestStatus = 0
	JoinRequestApproved JoinRequestStatus = 1
	JoinRequestRejected JoinRequestStatus = 2
)

type JoinRequest struct {
	ConvID    string            `json:"conv_id"`
	UserID    string            `json:"user_id"`
	Status    JoinRequestStatus `json:"status"`
	CreatedAt int64             `json:"created_at"`
	UpdatedAt int64             `json:"updated_at"`
}
