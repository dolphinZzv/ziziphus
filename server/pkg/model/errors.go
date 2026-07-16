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
	ErrConflict     = 4006
	ErrInternal     = 5001
)

func NewAppError(code int, msg string) *AppError {
	return &AppError{Code: code, Message: msg}
}

var (
	ErrBadMsgContent    = &AppError{Code: ErrBadMessage, Message: "illegal message content", Key: "err.bad_msg_content"}
	ErrConvNotFound     = &AppError{Code: ErrNotFound, Message: "conversation not found or no permission", Key: "err.conv_not_found"}
	ErrRateLimited      = &AppError{Code: ErrRateLimit, Message: "rate limited", Key: "err.rate_limited"}
	ErrNotInConv        = &AppError{Code: ErrNoPermission, Message: "sender not in conversation", Key: "err.not_in_conv"}
	ErrMsgTooLarge      = &AppError{Code: ErrTooLarge, Message: "message body too large", Key: "err.msg_too_large"}
	ErrInternalServer   = &AppError{Code: ErrInternal, Message: "internal server error", Key: "err.internal_server"}
	ErrDuplicateRequest = &AppError{Code: ErrBadMessage, Message: "pending join request already exists", Key: "err.duplicate_join_request"}
	ErrAlreadyMember    = &AppError{Code: ErrBadMessage, Message: "already a group member", Key: "err.already_member"}
	ErrNoPendingRequest = &AppError{Code: ErrNotFound, Message: "no pending request", Key: "err.no_pending_request"}

	// Friend request errors
	ErrContactRequestSelf           = &AppError{Code: ErrBadMessage, Message: "cannot send friend request to self", Key: "err.contact_request_self"}
	ErrContactRequestDuplicate      = &AppError{Code: ErrConflict, Message: "pending friend request already exists", Key: "err.contact_request_duplicate"}
	ErrAlreadyFriends               = &AppError{Code: ErrConflict, Message: "already friends", Key: "err.contact_request_already_friends"}
	ErrContactRequestNotFound       = &AppError{Code: ErrNotFound, Message: "friend request not found", Key: "err.contact_request_not_found"}
	ErrContactRequestAlreadyHandled = &AppError{Code: ErrConflict, Message: "request already processed", Key: "err.contact_request_already_handled"}

	// Ban errors
	ErrUserBanned = &AppError{Code: ErrNoPermission, Message: "account has been banned", Key: "err.user_banned"}
	ErrBanSelf    = &AppError{Code: ErrBadMessage, Message: "cannot ban yourself", Key: "err.ban_self"}
)
