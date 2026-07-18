package i18n

func init() {
	registerLang(LangJA, map[string]string{
		// Auth
		"auth.account_exists": "アカウントは既に存在します",
		"auth.bad_credentials": "アカウントまたはパスワードが無効です",
		"auth.invalid_refresh_token": "リフレッシュトークンが無効か期限切れです",
		"auth.password_too_short": "パスワードは8文字以上必要です",
		"auth.user_not_found": "ユーザーが見つかりません",
		"auth.wrong_password": "パスワードが間違っています",
		"auth.reset_user_not_found": "アカウントまたはメールが見つかりません",
		"auth.reset_no_email": "メールアドレスが登録されていません",
		"auth.reset_code_invalid": "リセットコードが無効または期限切れです",
		"auth.reset_code_expired": "リセットコードの期限が切れました",
		"err.mailer_disabled": "メールサービスが利用できません",
		"err.agent_limit": "エージェントの上限に達しました（最大10個）",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "メッセージ内容が無効です",
		"err.conv_not_found": "会話が見つからないか、権限がありません",
		"err.internal_server": "サーバー内部エラー",
		"err.msg_too_large": "メッセージ本文が大きすぎます",
		"err.not_in_conv": "送信者は会話に参加していません",
		"err.rate_limited": "メッセージのレート制限を超過しました",
		// HTTP Response helpers
		"err.resource_not_found": "リソースが見つかりません",
		"err.unauthorized": "認証されていません",
		// Handler validation
		"err.cannot_add_self": "自分自身を連絡先に追加することはできません",
		"err.cannot_chat_self": "自分自身とチャットすることはできません",
		"err.contact_user_id_required": "ユーザーIDは必須です",
		"err.group_only": "グループのみ対応",
		"err.invalid_email": "メールアドレスの形式が無効です",
		"err.invalid_params": "無効なパラメータです",
		"err.name_password_required": "ニックネームとパスワードは必須です",
		"err.name_required": "グループ名は必須です",
		"err.not_in_conv_specific": "この会話に参加していません",
		"err.registration_disabled": "新規登録は無効になっています",
		"err.user_id_required": "ユーザーIDは必須です",
		// Conversation manager
		"err.agent_owner_only": "Agentの作成者のみがグループに追加できます",
		"err.already_member": "既にグループメンバーです",
		"err.conv_not_found_mgr": "会話が見つかりません",
		"err.direct_chat_disabled": "このユーザーは直接チャットを無効にしています",
		"err.duplicate_join_request": "保留中の参加リクエストが既に存在します",
		"err.group_full": "グループのメンバー上限に達しました",
		"err.no_pending_request": "保留中の参加リクエストはありません",
		"err.owner_only": "グループの所有者だけがグループを解散できます",
		"err.permission_denied": "権限がありません",
		"err.user_not_found": "ユーザーが見つかりません",
		// WS handler
		"err.create_session_failed": "セッションの作成に失敗しました",
		// System notification messages
		"sys.file_deleted": "%s がファイルを削除しました: %s",
		"sys.file_uploaded": "%s がファイルをアップロードしました: %s",
		"sys.group_created": "%s がグループ「%s」を作成しました",
		"sys.member_added": "%s がグループに参加しました",
		"sys.member_left": "%s がグループを退出しました",
		"sys.member_removed": "%s がグループから削除されました",
		// Friend request messages
		"contact_request.already_handled": "このリクエストは既に処理されています",
		"contact_request.approve": "承認",
		"contact_request.approved": "承認済み",
		"contact_request.approved_by": "%s があなたの友達リクエストを承認しました",
		"contact_request.friend_established": "友達になりました。チャットを始めましょう！",
		"contact_request.reject": "拒否",
		"contact_request.rejected": "拒否済み",
		"contact_request.rejected_by": "%s があなたの友達リクエストを拒否しました",
		"contact_request.sent": "%s に友達リクエストを送信しました",
		"contact_request.title": "友達リクエスト",
		"contact_request.you_approved": "%s の友達リクエストを承認しました",
		"contact_request.you_rejected": "%s の友達リクエストを拒否しました",
		// MFA
		"err.mfa_email_required": "メール2FAを有効にするにはメールアドレスが必要です",
		"err.mfa_invalid_code": "無効な確認コードです",
		"err.mfa_not_found": "MFAが設定されていません",
		// Friend request errors
		"err.contact_request_already_friends": "既に友達です",
		"err.contact_request_already_handled": "このリクエストは既に処理されています",
		"err.contact_request_duplicate": "保留中の友達リクエストが既に存在します",
		"err.contact_request_not_found": "友達リクエストが見つかりません",
		"err.contact_request_self": "自分自身に友達リクエストを送信できません",
		// Ban errors
		"err.ban_self": "自分自身を停止することはできません",
		"err.user_banned": "アカウントは停止されました",
	})
}
