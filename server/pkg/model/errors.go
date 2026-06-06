package model

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

const (
	ErrBadMessage    = 4001
	ErrNoPermission  = 4002
	ErrRateLimit     = 4003
	ErrNotFound      = 4004
	ErrTooLarge      = 4005
	ErrInternal      = 5001
)

func NewAppError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

var (
	ErrBadMsgContent  = &AppError{Code: ErrBadMessage, Message: "消息内容非法"}
	ErrConvNotFound   = &AppError{Code: ErrNotFound, Message: "会话不存在或无权限"}
	ErrRateLimited    = &AppError{Code: ErrRateLimit, Message: "消息频率超限"}
	ErrNotInConv      = &AppError{Code: ErrNoPermission, Message: "发送者不在会话中"}
	ErrMsgTooLarge    = &AppError{Code: ErrTooLarge, Message: "消息体过大"}
	ErrInternalServer = &AppError{Code: ErrInternal, Message: "服务端内部错误"}
)
