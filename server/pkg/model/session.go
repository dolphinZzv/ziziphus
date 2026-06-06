package model

type SessionStatus int

const (
	SessionActive   SessionStatus = 0
	SessionInactive SessionStatus = 1
	SessionExpired  SessionStatus = 2
)

type DeviceType int

const (
	DevicePhone  DeviceType = 0
	DeviceDesktop DeviceType = 1
	DeviceWeb    DeviceType = 2
)

type Session struct {
	SessionID  string        `json:"session_id"`
	UserID     string        `json:"user_id"`
	Device     DeviceType    `json:"device"`
	DeviceName string        `json:"device_name"`
	ConnID     string        `json:"conn_id,omitempty"`
	Status     SessionStatus `json:"status"`
	LoginAt    int64         `json:"login_at"`
	LastActive int64         `json:"last_active"`
	Metadata   map[string]any        `json:"metadata,omitempty"`
}
