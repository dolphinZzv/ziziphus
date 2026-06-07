package model

type UserType int

const (
	UserHuman UserType = 0
	UserAgent UserType = 1
)

type UserStatus int

const (
	UserOffline UserStatus = 0
	UserOnline  UserStatus = 1
	UserBusy    UserStatus = 2
)

type User struct {
	ID        string     `json:"user_id"`
	Account   string     `json:"account"`
	Type      UserType   `json:"type"`
	Name      string     `json:"name"`
	Avatar    string     `json:"avatar"`
	Status    UserStatus `json:"status"`
	Password  string     `json:"-"`
	ExtMeta   map[string]any    `json:"ext_meta,omitempty"`
	CreatedAt int64      `json:"created_at"`
}

type OnlineDevice struct {
	Device     DeviceType `json:"device"`
	DeviceName string     `json:"device_name"`
	LastActive int64      `json:"last_active"`
}
