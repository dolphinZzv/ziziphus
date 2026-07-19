package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
)

type Handlers struct {
	User         *UserHandler
	Conversation *ConvHandler
	Message      *MsgHandler
	Contact      *ContactHandler
	Session      *SessionHandler
	File         *FileHandler
	Webhook      *WebhookHandler
	OAuth        *OAuthHandler
	Announcement http.HandlerFunc
	AppInfo      http.HandlerFunc
	DB           *pgxpool.Pool
	RDB          *redis.Client
	LoginRL      *LoginRateLimiter
	RegisterRL   *RegisterLimiter
	GlobalRL     *GlobalRateLimiter
}

func NewRouter(h *Handlers, authMW func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Use(chimw.Recoverer)
	r.Use(securityHeaders)
	r.Use(i18n.Middleware)

	// Global per-IP rate limiting (DDoS deterrent). Excludes health/metrics.
	if h.GlobalRL != nil {
		r.Use(h.GlobalRL.Middleware)
	}

	// Public routes (login rate limited)
	r.Group(func(r chi.Router) {
		if h.LoginRL != nil {
			r.Use(h.LoginRL.Middleware)
		}
		if h.RegisterRL != nil {
			r.With(h.RegisterRL.Middleware).Post("/api/v1/users/register", h.User.Register)
		} else {
			r.Post("/api/v1/users/register", h.User.Register)
		}
		r.Post("/api/v1/users/login", h.User.Login)
		r.Post("/api/v1/users/refresh", h.User.Refresh)
		r.Get("/api/v1/version", h.GetVersion)
		r.Get("/health", h.Health)
		r.Get("/metrics", promhttp.Handler().ServeHTTP)
		r.Get("/swagger/*", httpSwagger.WrapHandler)
		r.Post("/api/v1/auth/mfa/verify", h.User.MFAVerifyLogin)
		r.Post("/api/v1/webhooks/receive", h.Webhook.ReceiveMessage)
		r.Get("/api/v1/auth/{provider}/login", h.OAuth.Login)
		r.Get("/api/v1/auth/{provider}/callback", h.OAuth.Callback)
	})

	// Public routes without login rate limiting
	r.Get("/api/v1/i18n/detect", h.DetectLanguage)
	r.Get("/api/v1/announcement", h.Announcement)
	r.Get("/api/v1/app/info", h.AppInfo)
	r.Post("/api/v1/users/password-reset/send-code", h.User.SendPasswordResetCode)
	r.Post("/api/v1/users/password-reset/reset", h.User.ResetPassword)
	r.Get("/api/v1/groups/card/{share_token}", h.Conversation.GetGroupCard)

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Get("/api/v1/users/me", h.User.GetMe)
		r.Get("/api/v1/users/{user_id}", h.User.GetUser)
		r.Post("/api/v1/users/batch", h.User.BatchGet)
		r.Put("/api/v1/users/me", h.User.UpdateMe)
		r.Get("/api/v1/users/me/mfa", h.User.GetMFA)
		r.Post("/api/v1/users/me/mfa/setup", h.User.SetupMFA)
		r.Post("/api/v1/users/me/mfa/verify", h.User.VerifyMFA)
		r.Post("/api/v1/users/me/mfa/disable", h.User.DisableMFA)
		r.Post("/api/v1/users/me/email/send-code", h.User.SendEmailCode)
		r.Post("/api/v1/users/me/email/confirm", h.User.ConfirmEmail)
		r.Get("/api/v1/users/search", h.User.Search)
		r.Get("/api/v1/users/me/agents", h.User.ListMyAgents)
		r.Post("/api/v1/users/me/agents", h.User.CreateAgent)
		r.Put("/api/v1/users/me/agents/{agent_id}", h.User.UpdateAgent)
		r.Delete("/api/v1/users/me/agents/{agent_id}", h.User.DeleteAgent)
		r.Delete("/api/v1/users/me", h.User.DeleteAccount)
		r.Get("/api/v1/users/me/export", h.User.ExportData)
		r.Put("/api/v1/users/me/agents/{agent_id}/regenerate-key", h.User.RegenerateAgentKey)
		r.Get("/api/v1/groups/search", h.Conversation.SearchGroups)

		r.Get("/api/v1/conversations", h.Conversation.List)
		r.Get("/api/v1/conversations/{conv_id}", h.Conversation.GetDetail)
		r.Post("/api/v1/conversations/group", h.Conversation.CreateGroup)
		r.Post("/api/v1/conversations/p2p", h.Conversation.CreateP2P)
		r.Put("/api/v1/conversations/{conv_id}", h.Conversation.UpdateGroup)
		r.Post("/api/v1/conversations/{conv_id}/members", h.Conversation.AddMembers)
		r.Delete("/api/v1/conversations/{conv_id}/members/{user_id}", h.Conversation.RemoveMember)
		r.Post("/api/v1/conversations/{conv_id}/leave", h.Conversation.Leave)
		r.Post("/api/v1/conversations/{conv_id}/disband", h.Conversation.Disband)
		r.Post("/api/v1/conversations/{conv_id}/read", h.Conversation.MarkRead)
		r.Post("/api/v1/conversations/{conv_id}/join-requests", h.Conversation.RequestJoin)
		r.Get("/api/v1/conversations/{conv_id}/join-requests", h.Conversation.ListJoinRequests)
		r.Post("/api/v1/conversations/{conv_id}/join-requests/{user_id}/approve", h.Conversation.ApproveJoinRequest)
		r.Post("/api/v1/conversations/{conv_id}/join-requests/{user_id}/reject", h.Conversation.RejectJoinRequest)
		r.Post("/api/v1/conversations/{conv_id}/pin", h.Conversation.Pin)
		r.Post("/api/v1/conversations/{conv_id}/unpin", h.Conversation.Unpin)
		r.Post("/api/v1/conversations/{conv_id}/clone", h.Conversation.Clone)
		r.Get("/api/v1/conversations/{conv_id}/settings", h.Conversation.GetSettings)
		r.Put("/api/v1/conversations/{conv_id}/settings", h.Conversation.UpdateSettings)
		r.Post("/api/v1/conversations/{conv_id}/share-token", h.Conversation.GenerateShareToken)
		r.Delete("/api/v1/conversations/{conv_id}/share-token", h.Conversation.RemoveShareToken)
		// Webhook management
		r.Get("/api/v1/conversations/{conv_id}/webhooks", h.Webhook.List)
		r.Post("/api/v1/conversations/{conv_id}/webhooks", h.Webhook.Create)
		r.Put("/api/v1/conversations/{conv_id}/webhooks/{webhook_id}", h.Webhook.Update)
		r.Delete("/api/v1/conversations/{conv_id}/webhooks/{webhook_id}", h.Webhook.Delete)
		r.Post("/api/v1/conversations/{conv_id}/webhooks/{webhook_id}/regenerate-key", h.Webhook.RegenerateKey)
		r.Post("/api/v1/conversations/{conv_id}/webhooks/{webhook_id}/test", h.Webhook.Test)
		r.Get("/api/v1/conversations/{conv_id}/files", h.File.ListConvFiles)
		r.Delete("/api/v1/conversations/{conv_id}/files/{file_id}", h.File.DeleteConvFile)
		r.Post("/api/v1/conversations/{conv_id}/folders", h.File.CreateFolder)
		r.Delete("/api/v1/conversations/{conv_id}/folders", h.File.DeleteFolder)
		r.Put("/api/v1/conversations/{conv_id}/files/{file_id}/move", h.File.MoveFile)
		r.Put("/api/v1/conversations/{conv_id}/folders/move", h.File.MoveFolder)
		r.Put("/api/v1/conversations/{conv_id}/folders/rename", h.File.RenameFolder)
		r.Get("/api/v1/conversations/{conv_id}/folders/files", h.File.ListFolderFiles)
		r.Get("/api/v1/conversations/{conv_id}/folders", h.File.ListFolders)
		r.Get("/api/v1/conversations/unread/total", h.Conversation.UnreadTotal)

		r.Get("/api/v1/conversations/{conv_id}/messages", h.Message.GetHistory)
		r.Get("/api/v1/messages/{msg_id}/receipts", h.Message.GetReceipts)

		r.Get("/api/v1/contacts", h.Contact.List)
		r.Post("/api/v1/contacts", h.Contact.Add)
		r.Delete("/api/v1/contacts/{user_id}", h.Contact.Remove)
		r.Put("/api/v1/contacts/{user_id}", h.Contact.UpdateNickname)

		// Friend requests
		r.Post("/api/v1/contact-requests", h.Contact.RequestContact)
		r.Get("/api/v1/contact-requests/sent", h.Contact.ListSentRequests)
		r.Get("/api/v1/contact-requests/received", h.Contact.ListReceivedRequests)
		r.Get("/api/v1/contact-requests/by-form/{msg_id}", h.Contact.GetRequestByFormMsgID)
		r.Get("/api/v1/sessions", h.Session.ListSessions)
		r.Delete("/api/v1/sessions/{session_id}", h.Session.DeleteSession)

		// File upload & access (authenticated)
		r.Post("/api/v1/files/upload", h.File.Upload)
		r.Get("/api/v1/files/{file_id}", h.File.GetInfo)
		r.Post("/api/v1/files/token", h.File.RenewFileToken)
	})

	// Static file serving (public)
	r.Get("/files/*", h.File.ServeFile)

	return r
}

// requestLogger is a custom logger that redacts sensitive query parameters
// (e.g. tokens) before logging the request URL.
func requestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Redact token from URL for logging
		sanitized := r.URL.Path
		if r.URL.RawQuery != "" {
			query := r.URL.Query()
			if query.Has("token") {
				query.Set("token", "REDACTED")
			}
			sanitized = r.URL.Path + "?" + query.Encode()
		}

		ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
		next.ServeHTTP(ww, r)

		latency := time.Since(start)
		logger.Info("request",
			"method", r.Method,
			"path", sanitized,
			"status", ww.Status(),
			"latency_ms", latency.Milliseconds(),
			"ip", r.RemoteAddr,
			"ua", r.UserAgent(),
		)
	})
}

// securityHeaders sets standard HTTP security headers.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		next.ServeHTTP(w, r)
	})
}
