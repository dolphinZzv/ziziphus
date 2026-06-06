package model

type ReceiptStatus int

const (
	ReceiptDelivered ReceiptStatus = 1
	ReceiptRead      ReceiptStatus = 2
)

type Receipt struct {
	MsgID     int64         `json:"msg_id"`
	UserID    string        `json:"user_id"`
	SessionID string        `json:"session_id"`
	Status    ReceiptStatus `json:"status"`
	Timestamp int64         `json:"timestamp"`
}
