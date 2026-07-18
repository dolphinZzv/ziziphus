package i18n

func init() {
	registerLang(LangFR, map[string]string{
		// Auth
		"auth.account_exists":        "Le compte existe déjà",
		"auth.bad_credentials":       "Compte ou mot de passe invalide",
		"auth.invalid_refresh_token": "Le jeton de rafraîchissement est invalide ou a expiré",
		"auth.password_too_short":    "Le mot de passe doit contenir au moins 8 caractères",
		"auth.user_not_found":        "Utilisateur introuvable",
		"auth.wrong_password":        "Mot de passe incorrect",
		"auth.reset_user_not_found":  "Compte ou email introuvable",
		"auth.reset_no_email":        "Aucun email enregistré",
		"auth.reset_code_invalid":    "Code de réinitialisation invalide ou expiré",
		"auth.reset_code_expired":    "Le code de réinitialisation a expiré",
		"err.mailer_disabled":        "Service de messagerie indisponible",
		"err.agent_limit":            "Limite d'agents atteinte (10 max)",
		// Errors from pkg/model/errors.go
		"err.bad_msg_content": "Contenu du message invalide",
		"err.conv_not_found":  "Conversation introuvable ou permission refusée",
		"err.internal_server": "Erreur interne du serveur",
		"err.msg_too_large":   "Le corps du message est trop volumineux",
		"err.not_in_conv":     "L'expéditeur n'est pas dans la conversation",
		"err.rate_limited":    "Limite de taux de messages dépassée",
		// HTTP Response helpers
		"err.resource_not_found": "Ressource introuvable",
		"err.unauthorized":       "Non autorisé",
		// Handler validation
		"err.cannot_add_self":          "Vous ne pouvez pas vous ajouter comme contact",
		"err.cannot_chat_self":         "Vous ne pouvez pas discuter avec vous-même",
		"err.contact_user_id_required": "user_id est requis",
		"err.group_only":               "Groupes uniquement",
		"err.invalid_email":            "Format d'email invalide",
		"err.invalid_params":           "Paramètres invalides",
		"err.name_password_required":   "Le surnom et le mot de passe sont requis",
		"err.name_required":            "Le nom du groupe est requis",
		"err.not_in_conv_specific":     "Pas dans cette conversation",
		"err.registration_disabled":    "L'inscription est désactivée",
		"err.user_id_required":         "user_id est requis",
		// Conversation manager
		"err.agent_owner_only":       "Seul le créateur de l'agent peut l'ajouter au groupe",
		"err.already_member":         "Déjà membre du groupe",
		"err.conv_not_found_mgr":     "Conversation introuvable",
		"err.direct_chat_disabled":   "Le chat direct est désactivé par cet utilisateur",
		"err.duplicate_join_request": "Une demande d'adhésion en attente existe déjà",
		"err.group_full":             "La limite de membres du groupe est atteinte",
		"err.no_pending_request":     "Aucune demande d'adhésion en attente",
		"err.owner_only":             "Seul le propriétaire du groupe peut dissoudre le groupe",
		"err.permission_denied":      "Permission refusée",
		"err.user_not_found":         "Utilisateur introuvable",
		// WS handler
		"err.create_session_failed": "Échec de la création de la session",
		// System notification messages
		"sys.file_deleted":   "%s a supprimé : %s",
		"sys.file_uploaded":  "%s a téléchargé : %s",
		"sys.group_created":  "%s a créé le groupe \"%s\"",
		"sys.member_added":   "%s a rejoint le groupe",
		"sys.member_left":    "%s a quitté le groupe",
		"sys.member_removed": "%s a été retiré du groupe",
		// Friend request messages
		"contact_request.already_handled":    "Cette demande a déjà été traitée",
		"contact_request.approve":            "Approuver",
		"contact_request.approved":           "Approuvé",
		"contact_request.approved_by":        "%s a approuvé votre demande d'ami",
		"contact_request.friend_established": "Vous êtes maintenant amis. Commencez à discuter !",
		"contact_request.reject":             "Refuser",
		"contact_request.rejected":           "Refusé",
		"contact_request.rejected_by":        "%s a refusé votre demande d'ami",
		"contact_request.sent":               "Vous avez envoyé une demande d'ami à %s",
		"contact_request.title":              "Demande d'ami",
		"contact_request.you_approved":       "Vous avez approuvé la demande d'ami de %s",
		"contact_request.you_rejected":       "Vous avez refusé la demande d'ami de %s",
		// MFA
		"err.mfa_email_required": "Un email est requis pour activer l'authentification 2FA par email",
		"err.mfa_invalid_code":   "Code de vérification invalide",
		"err.mfa_not_found":      "MFA non configuré",
		// Friend request errors
		"err.contact_request_already_friends": "Vous êtes déjà amis",
		"err.contact_request_already_handled": "Cette demande a déjà été traitée",
		"err.contact_request_duplicate":       "Une demande d'ami en attente existe déjà",
		"err.contact_request_not_found":       "Demande d'ami introuvable",
		"err.contact_request_self":            "Vous ne pouvez pas envoyer une demande d'ami à vous-même",
		// Ban errors
		"err.ban_self":    "Vous ne pouvez pas vous bannir vous-même",
		"err.user_banned": "Le compte a été banni",
	})
}
