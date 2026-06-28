package model

import "time"

type EmailVerify struct {
	UserID       string    `json:"user_id"`
	PendingEmail string    `json:"pending_email"`
	Code         string    `json:"-"`
	ExpiresAt    time.Time `json:"expires_at"`
}
