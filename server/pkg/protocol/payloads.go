package protocol

type MsgSendPayload struct {
	ConvID      string   `json:"conv_id"`
	ContentType int      `json:"content_type"`
	Body        string   `json:"body"`
	ReplyTo     int64    `json:"reply_to"`
	ClientSeq   int64    `json:"client_seq"`
	Mention     []string `json:"mention,omitempty"`
}

type MsgSendAckPayload struct {
	MsgID     int64 `json:"msg_id"`
	Timestamp int64 `json:"timestamp"`
	ClientSeq int64 `json:"client_seq"`
	Status    int   `json:"status"`
}

type MsgPushPayload struct {
	MsgID       int64    `json:"msg_id"`
	ConvID      string   `json:"conv_id"`
	SenderID    string   `json:"sender_id"`
	SenderName  string   `json:"sender_name"`
	ContentType int      `json:"content_type"`
	Body        string   `json:"body"`
	ReplyTo     int64    `json:"reply_to"`
	Mention     []string `json:"mention,omitempty"`
	Timestamp   int64    `json:"timestamp"`
	ConvSeq     int64    `json:"conv_seq"`
}

type MsgReceivedPayload struct {
	MsgID   int64  `json:"msg_id"`
	ConvID  string `json:"conv_id"`
	ConvSeq int64  `json:"conv_seq"`
}

type SyncReqPayload struct {
	ConvID      string `json:"conv_id"`
	LastConvSeq int64  `json:"last_conv_seq"`
	Limit       int    `json:"limit"`
}

type SyncMessage struct {
	MsgID       int64  `json:"msg_id"`
	SenderID    string `json:"sender_id"`
	SenderName  string `json:"sender_name"`
	ContentType int    `json:"content_type"`
	Body        string `json:"body"`
	Timestamp   int64  `json:"timestamp"`
	ConvSeq     int64  `json:"conv_seq"`
}

type SyncResPayload struct {
	ConvID   string        `json:"conv_id"`
	Messages []SyncMessage `json:"messages"`
	HasMore  bool          `json:"has_more"`
}

type MsgReadNotifyPayload struct {
	ConvID    string `json:"conv_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
	MsgID     int64  `json:"msg_id"`
	Timestamp int64  `json:"timestamp"`
}

type SessionEventPayload struct {
	UserID     string `json:"user_id"`
	SessionID  string `json:"session_id"`
	Device     int    `json:"device,omitempty"`
	DeviceName string `json:"device_name,omitempty"`
}

type SessionRecoverPayload struct {
	SessionID string `json:"session_id"`
}

type SessionRecoverAckPayload struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Timestamp int64  `json:"timestamp"`
}

type TypingPayload struct {
	ConvID    string `json:"conv_id"`
	UserID    string `json:"user_id"`
	SessionID string `json:"session_id"`
}

type MsgEditPayload struct {
	ConvID  string `json:"conv_id"`
	MsgID   int64  `json:"msg_id"`
	NewBody string `json:"new_body"`
}

type MsgRecallPayload struct {
	ConvID string `json:"conv_id"`
	MsgID  int64  `json:"msg_id"`
}

type MsgEditPushPayload struct {
	ConvID    string `json:"conv_id"`
	MsgID     int64  `json:"msg_id"`
	SenderID  string `json:"sender_id"`
	NewBody   string `json:"new_body"`
	EditedAt  int64  `json:"edited_at"`
	Timestamp int64  `json:"timestamp"`
}

type MsgRecallPushPayload struct {
	ConvID     string `json:"conv_id"`
	MsgID      int64  `json:"msg_id"`
	SenderID   string `json:"sender_id"`
	RecalledAt int64  `json:"recalled_at"`
	Timestamp  int64  `json:"timestamp"`
}

type EmptyPayload struct{}

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}
