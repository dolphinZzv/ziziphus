package model

type UserType int

const (
	UserHuman UserType = 0
	UserAgent UserType = 1
)

type WakeMode int

const (
	WakeModeAll     WakeMode = 0
	WakeModeMention WakeMode = 1
)

type UserStatus int

const (
	UserOffline UserStatus = 0
	UserOnline  UserStatus = 1
	UserBusy    UserStatus = 2
)

type User struct {
	ID              string         `json:"user_id"`
	Account         string         `json:"account"`
	Type            UserType       `json:"type"`
	Name            string         `json:"name"`
	Email           string         `json:"email,omitempty"`
	Avatar          string         `json:"avatar"`
	Cover           string         `json:"cover,omitempty"`
	Status          UserStatus     `json:"status"`
	Password        string         `json:"-"`
	UID             string         `json:"uid"`
	PrimaryColor    string         `json:"primary_color"`
	SecondaryColor  string         `json:"secondary_color"`
	Banned          bool           `json:"banned"`
	ExtMeta         map[string]any `json:"ext_meta,omitempty"`
	WakeMode        WakeMode       `json:"wake_mode"`
	APIKey          string         `json:"api_key"`
	Headline        string         `json:"headline,omitempty"`
	Language        string         `json:"language,omitempty"`
	Discoverable    bool           `json:"discoverable"`
	AllowDirectChat bool           `json:"allow_direct_chat"`
	ConvLimit       int            `json:"conv_limit"`
	CreatedAt       int64          `json:"created_at"`
}

type OnlineDevice struct {
	Device     DeviceType `json:"device"`
	DeviceName string     `json:"device_name"`
	LastActive int64      `json:"last_active"`
}
