package i18n

func init() {
	registerLang(LangES, map[string]string{
		// Auth
		"auth.account_exists": "La cuenta ya existe",
		"auth.bad_credentials": "Cuenta o contraseña inválida",
		"auth.invalid_refresh_token": "El token de actualización no es válido o ha expirado",
		"auth.password_too_short": "La contraseña debe tener al menos 8 caracteres",
		"auth.user_not_found": "Usuario no encontrado",
		"auth.wrong_password": "Contraseña incorrecta",
		"auth.reset_user_not_found": "Cuenta o correo no encontrado",
		"auth.reset_no_email": "No hay correo registrado",
		"auth.reset_code_invalid": "Código de restablecimiento inválido o expirado",
		"auth.reset_code_expired": "El código de restablecimiento ha expirado",
		"err.mailer_disabled": "Servicio de correo no disponible",
		"err.agent_limit": "Límite de agents alcanzado (máx. 10)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "Contenido de mensaje inválido",
		"err.conv_not_found": "Conversación no encontrada o sin permiso",
		"err.internal_server": "Error interno del servidor",
		"err.msg_too_large": "El cuerpo del mensaje es demasiado grande",
		"err.not_in_conv": "El remitente no está en la conversación",
		"err.rate_limited": "Límite de tasa de mensajes excedido",
		// HTTP Response helpers
		"err.resource_not_found": "Recurso no encontrado",
		"err.unauthorized": "No autorizado",
		// Handler validation
		"err.cannot_add_self": "No puedes agregarte como contacto",
		"err.cannot_chat_self": "No puedes chatear contigo mismo",
		"err.contact_user_id_required": "user_id es obligatorio",
		"err.group_only": "Solo grupos",
		"err.invalid_email": "Formato de correo electrónico inválido",
		"err.invalid_params": "Parámetros inválidos",
		"err.name_password_required": "El apodo y la contraseña son obligatorios",
		"err.name_required": "El nombre del grupo es obligatorio",
		"err.not_in_conv_specific": "No estás en esta conversación",
		"err.registration_disabled": "El registro está deshabilitado",
		"err.user_id_required": "user_id es obligatorio",
		// Conversation manager
		"err.agent_owner_only": "Solo el creador del agente puede agregarlo al grupo",
		"err.already_member": "Ya eres miembro del grupo",
		"err.conv_not_found_mgr": "Conversación no encontrada",
		"err.direct_chat_disabled": "El chat directo está deshabilitado por este usuario",
		"err.duplicate_join_request": "Ya existe una solicitud de unión pendiente",
		"err.group_full": "Límite de miembros del grupo alcanzado",
		"err.no_pending_request": "No hay solicitudes de unión pendientes",
		"err.owner_only": "Solo el propietario del grupo puede disolverlo",
		"err.permission_denied": "Permiso denegado",
		"err.user_not_found": "Usuario no encontrado",
		// WS handler
		"err.create_session_failed": "Error al crear la sesión",
		// System notification messages
		"sys.file_deleted": "%s eliminó: %s",
		"sys.file_uploaded": "%s subió: %s",
		"sys.group_created": "%s creó el grupo \"%s\"",
		"sys.member_added": "%s se unió al grupo",
		"sys.member_left": "%s abandonó el grupo",
		"sys.member_removed": "%s fue eliminado del grupo",
		// Friend request messages
		"contact_request.already_handled": "Esta solicitud ya ha sido procesada",
		"contact_request.approve": "Aprobar",
		"contact_request.approved": "Aprobado",
		"contact_request.approved_by": "%s aprobó tu solicitud de amistad",
		"contact_request.friend_established": "Ahora son amigos. ¡Empiecen a chatear!",
		"contact_request.reject": "Rechazar",
		"contact_request.rejected": "Rechazado",
		"contact_request.rejected_by": "%s rechazó tu solicitud de amistad",
		"contact_request.sent": "Enviaste una solicitud de amistad a %s",
		"contact_request.title": "Solicitud de amistad",
		"contact_request.you_approved": "Aprobaste la solicitud de amistad de %s",
		"contact_request.you_rejected": "Rechazaste la solicitud de amistad de %s",
		// MFA
		"err.mfa_email_required": "Se requiere correo electrónico para habilitar la 2FA por email",
		"err.mfa_invalid_code": "Código de verificación inválido",
		"err.mfa_not_found": "MFA no configurado",
		// Friend request errors
		"err.contact_request_already_friends": "Ya sois amigos",
		"err.contact_request_already_handled": "Esta solicitud ya ha sido procesada",
		"err.contact_request_duplicate": "Ya existe una solicitud de amistad pendiente",
		"err.contact_request_not_found": "Solicitud de amistad no encontrada",
		"err.contact_request_self": "No puedes enviarte una solicitud de amistad a ti mismo",
		// Ban errors
		"err.ban_self": "No puedes bloquearte a ti mismo",
		"err.user_banned": "La cuenta ha sido bloqueada",
	})
}
