package protocol

import "encoding/json"

type MessageType int

const (
	MsgSend          MessageType = 1
	MsgSendAck       MessageType = 2
	MsgPush          MessageType = 11
	MsgReceived       MessageType = 12
	SyncReq          MessageType = 21
	SyncRes          MessageType = 22
	MsgReadNotify    MessageType = 32
	SessionOnline    MessageType = 41
	SessionOffline   MessageType = 42
	SessionRecover   MessageType = 43
	SessionRecoverAck MessageType = 44
	Typing           MessageType = 51
	Ping             MessageType = 61
	Pong             MessageType = 62
	Error            MessageType = 71
)

type Frame struct {
	Type    MessageType     `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}
