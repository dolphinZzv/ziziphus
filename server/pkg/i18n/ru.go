package i18n

func init() {
	registerLang(LangRU, map[string]string{
		// Auth
		"auth.account_exists":        "Аккаунт уже существует",
		"auth.bad_credentials":       "Неверный аккаунт или пароль",
		"auth.invalid_refresh_token": "Токен обновления недействителен или истёк",
		"auth.password_too_short":    "Пароль должен содержать не менее 8 символов",
		"auth.user_not_found":        "Пользователь не найден",
		"auth.wrong_password":        "Неверный пароль",
		"auth.reset_user_not_found":  "Аккаунт или email не найден",
		"auth.reset_no_email":        "Email не указан",
		"auth.reset_code_invalid":    "Недействительный или истёкший код сброса",
		"auth.reset_code_expired":    "Срок действия кода сброса истёк",
		"err.mailer_disabled":        "Почтовая служба недоступна",
		"err.agent_limit":            "Достигнут лимит агентов (макс. 10)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "Недопустимое содержимое сообщения",
		"err.conv_not_found":  "Беседа не найдена или нет разрешения",
		"err.internal_server": "Внутренняя ошибка сервера",
		"err.msg_too_large":   "Тело сообщения слишком большое",
		"err.not_in_conv":     "Отправитель не находится в беседе",
		"err.rate_limited":    "Превышен лимит частоты сообщений",
		// HTTP Response helpers
		"err.resource_not_found": "Ресурс не найден",
		"err.unauthorized":       "Не авторизован",
		// Handler validation
		"err.cannot_add_self":          "Нельзя добавить себя в контакты",
		"err.cannot_chat_self":         "Нельзя общаться с самим собой",
		"err.contact_user_id_required": "user_id обязателен",
		"err.group_only":               "Только для групп",
		"err.invalid_email":            "Неверный формат email",
		"err.invalid_params":           "Недопустимые параметры",
		"err.name_password_required":   "Никнейм и пароль обязательны",
		"err.name_required":            "Название группы обязательно",
		"err.not_in_conv_specific":     "Не в этой беседе",
		"err.registration_disabled":    "Регистрация отключена",
		"err.user_id_required":         "user_id обязателен",
		// Conversation manager
		"err.agent_owner_only":       "Только создатель агента может добавить его в группу",
		"err.already_member":         "Уже участник группы",
		"err.conv_not_found_mgr":     "Беседа не найдена",
		"err.direct_chat_disabled":   "Прямой чат отключён этим пользователем",
		"err.duplicate_join_request": "Уже есть ожидающий запрос на вступление",
		"err.group_full":             "Достигнут лимит участников группы",
		"err.no_pending_request":     "Нет ожидающих запросов на вступление",
		"err.owner_only":             "Только владелец группы может расформировать группу",
		"err.permission_denied":      "Доступ запрещён",
		"err.user_not_found":         "Пользователь не найден",
		// WS handler
		"err.create_session_failed": "Не удалось создать сессию",
		// System notification messages
		"sys.file_deleted":   "%s удалил(а) файл: %s",
		"sys.file_uploaded":  "%s загрузил(а) файл: %s",
		"sys.group_created":  "%s создал(а) группу \"%s\"",
		"sys.member_added":   "%s присоединился(ась) к группе",
		"sys.member_left":    "%s покинул(а) группу",
		"sys.member_removed": "%s был(а) удалён(а) из группы",
		// Friend request messages
		"contact_request.already_handled":    "Этот запрос уже был обработан",
		"contact_request.approve":            "Одобрить",
		"contact_request.approved":           "Одобрено",
		"contact_request.approved_by":        "%s одобрил(а) ваш запрос в друзья",
		"contact_request.friend_established": "Теперь вы друзья. Можете начать общение!",
		"contact_request.reject":             "Отклонить",
		"contact_request.rejected":           "Отклонено",
		"contact_request.rejected_by":        "%s отклонил(а) ваш запрос в друзья",
		"contact_request.sent":               "Вы отправили запрос в друзья %s",
		"contact_request.title":              "Запрос в друзья",
		"contact_request.you_approved":       "Вы одобрили запрос в друзья от %s",
		"contact_request.you_rejected":       "Вы отклонили запрос в друзья от %s",
		// MFA
		"err.mfa_email_required": "Требуется email для включения 2FA по email",
		"err.mfa_invalid_code":   "Неверный код подтверждения",
		"err.mfa_not_found":      "MFA не настроен",
		// Friend request errors
		"err.contact_request_already_friends": "Вы уже друзья",
		"err.contact_request_already_handled": "Этот запрос уже был обработан",
		"err.contact_request_duplicate":       "Уже есть ожидающий запрос в друзья",
		"err.contact_request_not_found":       "Запрос в друзья не найден",
		"err.contact_request_self":            "Нельзя отправить запрос в друзья самому себе",
		// Ban errors
		"err.ban_self":    "Нельзя заблокировать самого себя",
		"err.user_banned": "Аккаунт был заблокирован",
	})
}
