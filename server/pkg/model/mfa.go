package model

type MFAType int

const (
	MFANone  MFAType = 0
	MFATOTP  MFAType = 1
	MFAEmail MFAType = 2
)

type UserMFA struct {
	UserID  string  `json:"user_id"`
	MFAType MFAType `json:"mfa_type"`
	Enabled bool    `json:"enabled"`
	Secret  string  `json:"-"`
}
