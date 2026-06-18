package model

type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Key     string // i18n lookup key; set by sentinel errors or service layer
}

func (e *AppError) Error() string {
	return e.Message
}

const (
	ErrBadMessage   = 4001
	ErrNoPermission = 4002
	ErrRateLimit    = 4003
	ErrNotFound     = 4004
	ErrTooLarge     = 4005
	ErrInternal     = 5001
)

func NewAppError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

var (
	ErrBadMsgContent    = &AppError{Code: ErrBadMessage, Message: "消息内容非法", Key: "err.bad_msg_content"}
	ErrConvNotFound     = &AppError{Code: ErrNotFound, Message: "会话不存在或无权限", Key: "err.conv_not_found"}
	ErrRateLimited      = &AppError{Code: ErrRateLimit, Message: "消息频率超限", Key: "err.rate_limited"}
	ErrNotInConv        = &AppError{Code: ErrNoPermission, Message: "发送者不在会话中", Key: "err.not_in_conv"}
	ErrMsgTooLarge      = &AppError{Code: ErrTooLarge, Message: "消息体过大", Key: "err.msg_too_large"}
	ErrInternalServer   = &AppError{Code: ErrInternal, Message: "服务端内部错误", Key: "err.internal_server"}
	ErrDuplicateRequest = &AppError{Code: ErrBadMessage, Message: "已存在待处理的入群申请", Key: "err.duplicate_join_request"}
	ErrAlreadyMember    = &AppError{Code: ErrBadMessage, Message: "已经是群成员", Key: "err.already_member"}
	ErrNoPendingRequest = &AppError{Code: ErrNotFound, Message: "没有待处理的申请", Key: "err.no_pending_request"}
)
