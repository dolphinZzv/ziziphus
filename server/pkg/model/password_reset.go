package model

import "time"

type PasswordReset struct {
	UserID    string    `json:"user_id"`
	Code      string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
}
