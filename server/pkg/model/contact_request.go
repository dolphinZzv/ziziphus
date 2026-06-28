package model

// ContactRequestStatus enumerates the states of a friend request.
type ContactRequestStatus int

const (
	ContactRequestPending  ContactRequestStatus = 0
	ContactRequestApproved ContactRequestStatus = 1
	ContactRequestRejected ContactRequestStatus = 2
)

// ContactRequest represents a row in the contact_requests table.
type ContactRequest struct {
	ID         int64                `json:"id"`
	FromUserID string               `json:"from_user_id"`
	ToUserID   string               `json:"to_user_id"`
	FormMsgID  int64                `json:"form_msg_id"`
	Status     ContactRequestStatus `json:"status"`
	Message    string               `json:"message"`
	CreatedAt  int64                `json:"created_at"`
	UpdatedAt  int64                `json:"updated_at"`
}
