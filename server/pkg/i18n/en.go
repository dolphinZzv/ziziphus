package i18n

func init() {
	registerLang(LangEN, map[string]string{
		// Auth
		"auth.account_exists":        "Account already exists",
		"auth.bad_credentials":       "Invalid account or password",
		"auth.invalid_refresh_token": "Refresh token is invalid or has expired",
		"auth.password_too_short":    "Password must be at least 8 characters",
		"auth.user_not_found":        "User not found",
		"auth.wrong_password":        "Wrong password",
		"auth.reset_user_not_found":  "Account or email not found",
		"auth.reset_no_email":        "No email on file",
		"auth.reset_code_invalid":    "Invalid or expired reset code",
		"auth.reset_code_expired":    "Reset code has expired",
		"err.mailer_disabled":        "Mail service not available",
		"err.agent_limit":            "Agent limit reached (max 10)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "Invalid message content",
		"err.conv_not_found":  "Conversation not found or no permission",
		"err.internal_server": "Internal server error",
		"err.msg_too_large":   "Message body too large",
		"err.not_in_conv":     "Sender is not in the conversation",
		"err.rate_limited":    "Message rate limit exceeded",
		// HTTP Response helpers
		"err.resource_not_found": "Resource not found",
		"err.unauthorized":       "Unauthorized",
		// Handler validation
		"err.cannot_add_self":          "Cannot add yourself as contact",
		"err.cannot_chat_self":         "Cannot chat with yourself",
		"err.contact_user_id_required": "user_id is required",
		"err.group_only":               "Groups only",
		"err.invalid_email":            "Invalid email format",
		"err.invalid_params":           "Invalid parameters",
		"err.name_password_required":   "Nickname and password are required",
		"err.name_required":            "Group name is required",
		"err.not_in_conv_specific":     "Not in this conversation",
		"err.registration_disabled":    "Registration is disabled",
		"err.user_id_required":         "user_id is required",
		// Conversation manager
		"err.agent_owner_only":       "Only the agent creator can add it to the group",
		"err.already_member":         "Already a group member",
		"err.conv_not_found_mgr":     "Conversation not found",
		"err.direct_chat_disabled":   "Direct chat disabled by this user",
		"err.duplicate_join_request": "A pending join request already exists",
		"err.group_full":             "Group member limit reached",
		"err.no_pending_request":     "No pending join request",
		"err.owner_only":             "Only the group owner can dismiss the group",
		"err.permission_denied":      "Permission denied",
		"err.user_not_found":         "User not found",
		// WS handler
		"err.create_session_failed": "Failed to create session",
		// System notification messages
		"sys.file_deleted":   "%s deleted: %s",
		"sys.file_uploaded":  "%s uploaded: %s",
		"sys.group_created":  "%s created the group \"%s\"",
		"sys.member_added":   "%s joined the group",
		"sys.member_left":    "%s left the group",
		"sys.member_removed": "%s was removed from the group",
		// Friend request messages
		"contact_request.already_handled":    "This request has already been handled",
		"contact_request.approve":            "Approve",
		"contact_request.approved":           "Approved",
		"contact_request.approved_by":        "%s approved your friend request",
		"contact_request.friend_established": "You are now friends. Start chatting!",
		"contact_request.reject":             "Reject",
		"contact_request.rejected":           "Rejected",
		"contact_request.rejected_by":        "%s rejected your friend request",
		"contact_request.sent":               "You sent a friend request to %s",
		"contact_request.title":              "Friend Request",
		"contact_request.you_approved":       "You approved %s's friend request",
		"contact_request.you_rejected":       "You rejected %s's friend request",
		// MFA
		"err.mfa_email_required": "Email is required to enable email 2FA",
		"err.mfa_invalid_code":   "Invalid verification code",
		"err.mfa_not_found":      "MFA not set up",
		// Friend request errors
		"err.contact_request_already_friends": "You are already friends",
		"err.contact_request_already_handled": "This request has already been handled",
		"err.contact_request_duplicate":       "A pending friend request already exists",
		"err.contact_request_not_found":       "Friend request not found",
		"err.contact_request_self":            "Cannot send friend request to yourself",
		// Ban errors
		"err.ban_self":    "Cannot ban yourself",
		"err.user_banned": "Account has been banned",
	})
}
