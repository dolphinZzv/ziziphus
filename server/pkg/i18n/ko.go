package i18n

func init() {
	registerLang(LangKO, map[string]string{
		// Auth
		"auth.account_exists":        "계정이 이미 존재합니다",
		"auth.bad_credentials":       "계정 또는 비밀번호가 잘못되었습니다",
		"auth.invalid_refresh_token": "갱신 토큰이 유효하지 않거나 만료되었습니다",
		"auth.password_too_short":    "비밀번호는 8자 이상이어야 합니다",
		"auth.user_not_found":        "사용자를 찾을 수 없습니다",
		"auth.wrong_password":        "비밀번호가 틀렸습니다",
		"auth.reset_user_not_found":  "계정 또는 이메일을 찾을 수 없습니다",
		"auth.reset_no_email":        "등록된 이메일이 없습니다",
		"auth.reset_code_invalid":    "재설정 코드가 유효하지 않거나 만료되었습니다",
		"auth.reset_code_expired":    "재설정 코드가 만료되었습니다",
		"err.mailer_disabled":        "메일 서비스를 사용할 수 없습니다",
		"err.agent_limit":            "Agent 한도에 도달했습니다 (최대 10개)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "잘못된 메시지 내용입니다",
		"err.conv_not_found":  "대화를 찾을 수 없거나 권한이 없습니다",
		"err.internal_server": "서버 내부 오류",
		"err.msg_too_large":   "메시지 본문이 너무 큽니다",
		"err.not_in_conv":     "발신자가 대화에 참여하고 있지 않습니다",
		"err.rate_limited":    "메시지 전송 제한을 초과했습니다",
		// HTTP Response helpers
		"err.resource_not_found": "리소스를 찾을 수 없습니다",
		"err.unauthorized":       "권한이 없습니다",
		// Handler validation
		"err.cannot_add_self":          "자기 자신을 연락처로 추가할 수 없습니다",
		"err.cannot_chat_self":         "자기 자신과 채팅할 수 없습니다",
		"err.contact_user_id_required": "사용자 ID가 필요합니다",
		"err.group_only":               "그룹 전용입니다",
		"err.invalid_email":            "이메일 형식이 올바르지 않습니다",
		"err.invalid_params":           "잘못된 매개변수입니다",
		"err.name_password_required":   "닉네임과 비밀번호는 필수입니다",
		"err.name_required":            "그룹 이름은 필수입니다",
		"err.not_in_conv_specific":     "이 대화에 참여하고 있지 않습니다",
		"err.registration_disabled":    "회원가입이 비활성화되었습니다",
		"err.user_id_required":         "사용자 ID가 필요합니다",
		// Conversation manager
		"err.agent_owner_only":       "에이전트 생성자만 그룹에 추가할 수 있습니다",
		"err.already_member":         "이미 그룹 멤버입니다",
		"err.conv_not_found_mgr":     "대화를 찾을 수 없습니다",
		"err.direct_chat_disabled":   "이 사용자가 직접 채팅을 비활성화했습니다",
		"err.duplicate_join_request": "보류 중인 가입 요청이 이미 존재합니다",
		"err.group_full":             "그룹 인원 제한에 도달했습니다",
		"err.no_pending_request":     "보류 중인 가입 요청이 없습니다",
		"err.owner_only":             "그룹 소유자만 그룹을 해산할 수 있습니다",
		"err.permission_denied":      "권한이 거부되었습니다",
		"err.user_not_found":         "사용자를 찾을 수 없습니다",
		// WS handler
		"err.create_session_failed": "세션 생성에 실패했습니다",
		// System notification messages
		"sys.file_deleted":   "%s님이 파일을 삭제했습니다: %s",
		"sys.file_uploaded":  "%s님이 파일을 업로드했습니다: %s",
		"sys.group_created":  "%s님이 그룹 \"%s\"을(를) 만들었습니다",
		"sys.member_added":   "%s님이 그룹에 가입했습니다",
		"sys.member_left":    "%s님이 그룹을 나갔습니다",
		"sys.member_removed": "%s님이 그룹에서 제거되었습니다",
		// Friend request messages
		"contact_request.already_handled":    "이 요청은 이미 처리되었습니다",
		"contact_request.approve":            "승인",
		"contact_request.approved":           "승인됨",
		"contact_request.approved_by":        "%s님이 친구 요청을 승인했습니다",
		"contact_request.friend_established": "이제 친구입니다. 채팅을 시작하세요!",
		"contact_request.reject":             "거절",
		"contact_request.rejected":           "거절됨",
		"contact_request.rejected_by":        "%s님이 친구 요청을 거절했습니다",
		"contact_request.sent":               "%s님에게 친구 요청을 보냈습니다",
		"contact_request.title":              "친구 요청",
		"contact_request.you_approved":       "%s님의 친구 요청을 승인했습니다",
		"contact_request.you_rejected":       "%s님의 친구 요청을 거절했습니다",
		// MFA
		"err.mfa_email_required": "이메일 2FA를 활성화하려면 이메일이 필요합니다",
		"err.mfa_invalid_code":   "잘못된 인증 코드입니다",
		"err.mfa_not_found":      "MFA가 설정되지 않았습니다",
		// Friend request errors
		"err.contact_request_already_friends": "이미 친구입니다",
		"err.contact_request_already_handled": "이 요청은 이미 처리되었습니다",
		"err.contact_request_duplicate":       "보류 중인 친구 요청이 이미 존재합니다",
		"err.contact_request_not_found":       "친구 요청을 찾을 수 없습니다",
		"err.contact_request_self":            "자기 자신에게 친구 요청을 보낼 수 없습니다",
		// Ban errors
		"err.ban_self":    "자기 자신을 차단할 수 없습니다",
		"err.user_banned": "계정이 차단되었습니다",
	})
}
