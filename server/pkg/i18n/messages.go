package i18n

// Messages is the centralized message map for all server i18n.
// Keys follow a semantic dot-notation convention.
var Messages = map[string]map[Lang]string{
	// ===== Auth =====
	"auth.account_exists": {LangZH: "账户已存在", LangEN: "Account already exists"},
	"auth.user_not_found": {LangZH: "用户不存在", LangEN: "User not found"},
	"auth.wrong_password": {LangZH: "密码错误", LangEN: "Wrong password"},

	// ===== Errors from pkg/model/errors.go =====
	"err.bad_msg_content": {LangZH: "消息内容非法", LangEN: "Invalid message content"},
	"err.conv_not_found":  {LangZH: "会话不存在或无权限", LangEN: "Conversation not found or no permission"},
	"err.rate_limited":    {LangZH: "消息频率超限", LangEN: "Message rate limit exceeded"},
	"err.not_in_conv":     {LangZH: "发送者不在会话中", LangEN: "Sender is not in the conversation"},
	"err.msg_too_large":   {LangZH: "消息体过大", LangEN: "Message body too large"},
	"err.internal_server": {LangZH: "服务端内部错误", LangEN: "Internal server error"},

	// ===== HTTP Response helpers =====
	"err.unauthorized":       {LangZH: "未授权", LangEN: "Unauthorized"},
	"err.resource_not_found": {LangZH: "资源不存在", LangEN: "Resource not found"},

	// ===== Handler validation =====
	"err.invalid_params":           {LangZH: "参数错误", LangEN: "Invalid parameters"},
	"err.name_password_required":   {LangZH: "昵称和密码不能为空", LangEN: "Nickname and password are required"},
	"err.user_id_required":         {LangZH: "user_id 不能为空", LangEN: "user_id is required"},
	"err.name_required":            {LangZH: "群组名称不能为空", LangEN: "Group name is required"},
	"err.cannot_chat_self":         {LangZH: "不能和自己聊天", LangEN: "Cannot chat with yourself"},
	"err.not_in_conv_specific":     {LangZH: "不在会话中", LangEN: "Not in this conversation"},
	"err.group_only":               {LangZH: "仅支持群组", LangEN: "Groups only"},
	"err.cannot_add_self":          {LangZH: "不能添加自己为联系人", LangEN: "Cannot add yourself as contact"},
	"err.contact_user_id_required": {LangZH: "user_id 不能为空", LangEN: "user_id is required"},

	// ===== Conversation manager =====
	"err.conv_not_found_mgr":     {LangZH: "会话不存在", LangEN: "Conversation not found"},
	"err.permission_denied":      {LangZH: "权限不足", LangEN: "Permission denied"},
	"err.direct_chat_disabled":   {LangZH: "对方已关闭直接发起会话", LangEN: "Direct chat disabled by this user"},
	"err.user_not_found":         {LangZH: "用户不存在", LangEN: "User not found"},
	"err.group_full":             {LangZH: "群组人数已达上限", LangEN: "Group member limit reached"},
	"err.duplicate_join_request": {LangZH: "已存在待处理的入群申请", LangEN: "A pending join request already exists"},
	"err.already_member":         {LangZH: "已经是群成员", LangEN: "Already a group member"},
	"err.no_pending_request":     {LangZH: "没有待处理的申请", LangEN: "No pending join request"},

	// ===== WS handler =====
	"err.create_session_failed": {LangZH: "创建会话失败", LangEN: "Failed to create session"},

	// ===== System notification messages =====
	"sys.group_created":  {LangZH: "%s 创建了群「%s」", LangEN: "%s created the group \"%s\""},
	"sys.member_added":   {LangZH: "%s 被加入群", LangEN: "%s joined the group"},
	"sys.member_removed": {LangZH: "%s 被移出群", LangEN: "%s was removed from the group"},
	"sys.member_left":    {LangZH: "%s 退出了群", LangEN: "%s left the group"},

	// ===== Friend request messages =====
	"contact_request.title":              {LangZH: "好友申请", LangEN: "Friend Request"},
	"contact_request.approve":            {LangZH: "通过", LangEN: "Approve"},
	"contact_request.reject":             {LangZH: "拒绝", LangEN: "Reject"},
	"contact_request.approved":           {LangZH: "已通过", LangEN: "Approved"},
	"contact_request.rejected":           {LangZH: "已拒绝", LangEN: "Rejected"},
	"contact_request.sent":               {LangZH: "你已向 %s 发送了好友申请", LangEN: "You sent a friend request to %s"},
	"contact_request.approved_by":        {LangZH: "%s 已通过你的好友申请", LangEN: "%s approved your friend request"},
	"contact_request.rejected_by":        {LangZH: "%s 已拒绝你的好友申请", LangEN: "%s rejected your friend request"},
	"contact_request.you_approved":       {LangZH: "你已通过 %s 的好友申请", LangEN: "You approved %s's friend request"},
	"contact_request.you_rejected":       {LangZH: "你已拒绝 %s 的好友申请", LangEN: "You rejected %s's friend request"},
	"contact_request.friend_established": {LangZH: "你们已成为好友，可以开始聊天了", LangEN: "You are now friends. Start chatting!"},
	"contact_request.already_handled":    {LangZH: "该申请已被处理", LangEN: "This request has already been handled"},

	// ===== MFA =====
	"err.mfa_email_required": {LangZH: "开启邮箱认证需要设置邮箱", LangEN: "Email is required to enable email 2FA"},
	"err.mfa_invalid_code":     {LangZH: "验证码无效", LangEN: "Invalid verification code"},
	"err.mfa_not_found":        {LangZH: "未设置 MFA", LangEN: "MFA not set up"},

	// ===== Friend request errors =====
	"err.contact_request_self":             {LangZH: "不能给自己发好友申请", LangEN: "Cannot send friend request to yourself"},
	"err.contact_request_duplicate":        {LangZH: "已有待处理的好友申请", LangEN: "A pending friend request already exists"},
	"err.contact_request_already_friends":  {LangZH: "你们已经是好友", LangEN: "You are already friends"},
	"err.contact_request_not_found":        {LangZH: "好友申请不存在", LangEN: "Friend request not found"},
	"err.contact_request_already_handled":  {LangZH: "该申请已被处理", LangEN: "This request has already been handled"},
}
