package model

type ContentType int

const (
	ContentText   ContentType = 0
	ContentSystem ContentType = 5
)

type MsgStatus int

const (
	MsgSending   MsgStatus = 0
	MsgSent      MsgStatus = 1
	MsgDelivered MsgStatus = 2
	MsgRead      MsgStatus = 3
)

type Message struct {
	MsgID          int64      `json:"msg_id"`
	ConvID         string     `json:"conv_id"`
	SenderID       string     `json:"sender_id"`
	SenderName     string     `json:"sender_name"`
	SenderSessionID string    `json:"sender_session_id"`
	ContentType    ContentType `json:"content_type"`
	Body           string     `json:"body"`
	Mention        []string   `json:"mention,omitempty"`
	ReplyTo        int64      `json:"reply_to"`
	Timestamp      int64      `json:"timestamp"`
	ClientSeq      int64      `json:"client_seq"`
	ConvSeq        int64      `json:"conv_seq"`
	Status         MsgStatus  `json:"status"`
	Deleted        bool       `json:"deleted,omitempty"`
}
