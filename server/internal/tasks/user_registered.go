package tasks

import "encoding/json"

const TypeUserRegistered = "user_registered"

type UserRegisteredPayload struct {
	UserID string `json:"user_id"`
	Lang   string `json:"lang"`
	Email  string `json:"email,omitempty"`
}

func NewUserRegisteredTask(userID, lang, email string) ([]byte, error) {
	return json.Marshal(UserRegisteredPayload{UserID: userID, Lang: lang, Email: email})
}
