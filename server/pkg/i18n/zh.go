package i18n

func init() {
	registerLang(LangZH, map[string]string{
		// Auth
		"auth.account_exists":        "账户已存在",
		"auth.bad_credentials":       "账号或密码错误",
		"auth.invalid_refresh_token": "刷新令牌无效或已过期",
		"auth.password_too_short":    "密码长度不能少于8位",
		"auth.user_not_found":        "用户不存在",
		"auth.wrong_password":        "密码错误",
		"auth.reset_user_not_found":  "账号或邮箱未找到",
		"auth.reset_no_email":        "未设置邮箱",
		"auth.reset_code_invalid":    "验证码无效或已过期",
		"auth.reset_code_expired":    "验证码已过期",
		"err.mailer_disabled":        "邮件服务不可用",
		"err.agent_limit":            "Agent 数量已达上限（最多 10 个）",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "消息内容非法",
		"err.conv_not_found":  "会话不存在或无权限",
		"err.internal_server": "服务端内部错误",
		"err.msg_too_large":   "消息体过大",
		"err.not_in_conv":     "发送者不在会话中",
		"err.rate_limited":    "消息频率超限",
		// HTTP Response helpers
		"err.resource_not_found": "资源不存在",
		"err.unauthorized":       "未授权",
		// Handler validation
		"err.cannot_add_self":          "不能添加自己为联系人",
		"err.cannot_chat_self":         "不能和自己聊天",
		"err.contact_user_id_required": "user_id 不能为空",
		"err.group_only":               "仅支持群组",
		"err.invalid_email":            "邮箱格式无效",
		"err.invalid_params":           "参数错误",
		"err.name_password_required":   "昵称和密码不能为空",
		"err.name_required":            "群组名称不能为空",
		"err.not_in_conv_specific":     "不在会话中",
		"err.registration_disabled":    "新用户注册已关闭",
		"err.user_id_required":         "user_id 不能为空",
		// Conversation manager
		"err.agent_owner_only":       "只有 Agent 的创建者可以将其加入群组",
		"err.already_member":         "已经是群成员",
		"err.conv_not_found_mgr":     "会话不存在",
		"err.direct_chat_disabled":   "对方已关闭直接发起会话",
		"err.duplicate_join_request": "已存在待处理的入群申请",
		"err.group_full":             "群组人数已达上限",
		"err.no_pending_request":     "没有待处理的申请",
		"err.owner_only":             "只有群主可以解散群组",
		"err.permission_denied":      "权限不足",
		"err.user_not_found":         "用户不存在",
		// WS handler
		"err.create_session_failed": "创建会话失败",
		// System notification messages
		"sys.file_deleted":   "%s 删除了文件: %s",
		"sys.file_uploaded":  "%s 上传了文件: %s",
		"sys.group_created":  "%s 创建了群「%s」",
		"sys.member_added":   "%s 被加入群",
		"sys.member_left":    "%s 退出了群",
		"sys.member_removed": "%s 被移出群",
		// Friend request messages
		"contact_request.already_handled":    "该申请已被处理",
		"contact_request.approve":            "通过",
		"contact_request.approved":           "已通过",
		"contact_request.approved_by":        "%s 已通过你的好友申请",
		"contact_request.friend_established": "你们已成为好友，可以开始聊天了",
		"contact_request.reject":             "拒绝",
		"contact_request.rejected":           "已拒绝",
		"contact_request.rejected_by":        "%s 已拒绝你的好友申请",
		"contact_request.sent":               "你已向 %s 发送了好友申请",
		"contact_request.title":              "好友申请",
		"contact_request.you_approved":       "你已通过 %s 的好友申请",
		"contact_request.you_rejected":       "你已拒绝 %s 的好友申请",
		// MFA
		"err.mfa_email_required": "开启邮箱认证需要设置邮箱",
		"err.mfa_invalid_code":   "验证码无效",
		"err.mfa_not_found":      "未设置 MFA",
		// Friend request errors
		"err.contact_request_already_friends": "你们已经是好友",
		"err.contact_request_already_handled": "该申请已被处理",
		"err.contact_request_duplicate":       "已有待处理的好友申请",
		"err.contact_request_not_found":       "好友申请不存在",
		"err.contact_request_self":            "不能给自己发好友申请",
		// Ban errors
		"err.ban_self":    "不能封禁自己",
		"err.user_banned": "账号已被封禁",
	})
}
