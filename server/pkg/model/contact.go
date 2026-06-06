package model

type Contact struct {
	UserID    string `json:"user_id"`
	ContactID string `json:"contact_id"`
	Nickname  string `json:"nickname,omitempty"`
	AddedAt   int64  `json:"added_at"`
}
