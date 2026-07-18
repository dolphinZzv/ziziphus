package i18n

func init() {
	registerLang(LangDE, map[string]string{
		// Auth
		"auth.account_exists":        "Konto existiert bereits",
		"auth.bad_credentials":       "Ungültiges Konto oder Passwort",
		"auth.invalid_refresh_token": "Refresh-Token ist ungültig oder abgelaufen",
		"auth.password_too_short":    "Passwort muss mindestens 8 Zeichen lang sein",
		"auth.user_not_found":        "Benutzer nicht gefunden",
		"auth.wrong_password":        "Falsches Passwort",
		"auth.reset_user_not_found":  "Konto oder E-Mail nicht gefunden",
		"auth.reset_no_email":        "Keine E-Mail hinterlegt",
		"auth.reset_code_invalid":    "Reset-Code ungültig oder abgelaufen",
		"auth.reset_code_expired":    "Reset-Code ist abgelaufen",
		"err.mailer_disabled":        "E-Mail-Dienst nicht verfügbar",
		"err.agent_limit":            "Agenten-Limit erreicht (max. 10)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "Ungültiger Nachrichteninhalt",
		"err.conv_not_found":  "Konversation nicht gefunden oder keine Berechtigung",
		"err.internal_server": "Interner Serverfehler",
		"err.msg_too_large":   "Nachrichtentext zu groß",
		"err.not_in_conv":     "Der Absender ist nicht in der Konversation",
		"err.rate_limited":    "Nachrichtenratenlimit überschritten",
		// HTTP Response helpers
		"err.resource_not_found": "Ressource nicht gefunden",
		"err.unauthorized":       "Nicht autorisiert",
		// Handler validation
		"err.cannot_add_self":          "Sie können sich nicht selbst als Kontakt hinzufügen",
		"err.cannot_chat_self":         "Sie können nicht mit sich selbst chatten",
		"err.contact_user_id_required": "user_id ist erforderlich",
		"err.group_only":               "Nur Gruppen",
		"err.invalid_email":            "Ungültiges E-Mail-Format",
		"err.invalid_params":           "Ungültige Parameter",
		"err.name_password_required":   "Spitzname und Passwort sind erforderlich",
		"err.name_required":            "Gruppenname ist erforderlich",
		"err.not_in_conv_specific":     "Nicht in dieser Konversation",
		"err.registration_disabled":    "Registrierung ist deaktiviert",
		"err.user_id_required":         "user_id ist erforderlich",
		// Conversation manager
		"err.agent_owner_only":       "Nur der Agent-Ersteller kann ihn zur Gruppe hinzufügen",
		"err.already_member":         "Bereits Gruppenmitglied",
		"err.conv_not_found_mgr":     "Konversation nicht gefunden",
		"err.direct_chat_disabled":   "Direktchat wurde von diesem Benutzer deaktiviert",
		"err.duplicate_join_request": "Eine ausstehende Beitrittsanfrage existiert bereits",
		"err.group_full":             "Gruppenmitgliederlimit erreicht",
		"err.no_pending_request":     "Keine ausstehende Beitrittsanfrage",
		"err.owner_only":             "Nur der Gruppenbesitzer kann die Gruppe auflösen",
		"err.permission_denied":      "Zugriff verweigert",
		"err.user_not_found":         "Benutzer nicht gefunden",
		// WS handler
		"err.create_session_failed": "Fehler beim Erstellen der Sitzung",
		// System notification messages
		"sys.file_deleted":   "%s hat gelöscht: %s",
		"sys.file_uploaded":  "%s hat hochgeladen: %s",
		"sys.group_created":  "%s hat die Gruppe \"%s\" erstellt",
		"sys.member_added":   "%s ist der Gruppe beigetreten",
		"sys.member_left":    "%s hat die Gruppe verlassen",
		"sys.member_removed": "%s wurde aus der Gruppe entfernt",
		// Friend request messages
		"contact_request.already_handled":    "Diese Anfrage wurde bereits bearbeitet",
		"contact_request.approve":            "Genehmigen",
		"contact_request.approved":           "Genehmigt",
		"contact_request.approved_by":        "%s hat Ihre Freundschaftsanfrage genehmigt",
		"contact_request.friend_established": "Ihr seid jetzt Freunde. Beginnt mit dem Chatten!",
		"contact_request.reject":             "Ablehnen",
		"contact_request.rejected":           "Abgelehnt",
		"contact_request.rejected_by":        "%s hat Ihre Freundschaftsanfrage abgelehnt",
		"contact_request.sent":               "Sie haben eine Freundschaftsanfrage an %s gesendet",
		"contact_request.title":              "Freundschaftsanfrage",
		"contact_request.you_approved":       "Sie haben die Freundschaftsanfrage von %s genehmigt",
		"contact_request.you_rejected":       "Sie haben die Freundschaftsanfrage von %s abgelehnt",
		// MFA
		"err.mfa_email_required": "E-Mail ist erforderlich, um E-Mail-2FA zu aktivieren",
		"err.mfa_invalid_code":   "Ungültiger Bestätigungscode",
		"err.mfa_not_found":      "MFA nicht eingerichtet",
		// Friend request errors
		"err.contact_request_already_friends": "Ihr seid bereits Freunde",
		"err.contact_request_already_handled": "Diese Anfrage wurde bereits bearbeitet",
		"err.contact_request_duplicate":       "Eine ausstehende Freundschaftsanfrage existiert bereits",
		"err.contact_request_not_found":       "Freundschaftsanfrage nicht gefunden",
		"err.contact_request_self":            "Sie können keine Freundschaftsanfrage an sich selbst senden",
		// Ban errors
		"err.ban_self":    "Sie können sich nicht selbst sperren",
		"err.user_banned": "Konto wurde gesperrt",
	})
}
