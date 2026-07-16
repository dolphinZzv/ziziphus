package tasks

import "encoding/json"

const (
	TypeEmailVerification = "email:verification"
	TypePasswordReset     = "email:password_reset"
)

type EmailVerificationPayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
}

type PasswordResetPayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
}

func NewEmailVerificationTask(to, code string) ([]byte, error) {
	payload := EmailVerificationPayload{To: to, Code: code}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func NewPasswordResetTask(to, code string) ([]byte, error) {
	payload := PasswordResetPayload{To: to, Code: code}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return b, nil
}
