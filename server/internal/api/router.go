package api

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"siciv.space/agent/panda_ai/pkg/i18n"
)

type Handlers struct {
	User         *UserHandler
	Conversation *ConvHandler
	Message      *MsgHandler
	Contact      *ContactHandler
	Session      *SessionHandler
	File         *FileHandler
	DB           *pgxpool.Pool
	RDB          *redis.Client
	LoginRL      *LoginRateLimiter
}

func NewRouter(h *Handlers, authMW func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(requestLogger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(i18n.Middleware)

	// Public routes
	r.Group(func(r chi.Router) {
		// Apply login rate limiting if configured
		if h.LoginRL != nil {
			r.Use(h.LoginRL.Middleware)
		}
		r.Post("/api/v1/users/register", h.User.Register)
		r.Post("/api/v1/users/login", h.User.Login)
		r.Post("/api/v1/users/refresh", h.User.Refresh)
		r.Get("/api/v1/version", h.GetVersion)
		r.Get("/health", h.Health)
		r.Get("/metrics", promhttp.Handler().ServeHTTP)
	})

	// Authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Get("/api/v1/users/me", h.User.GetMe)
		r.Get("/api/v1/users/{user_id}", h.User.GetUser)
		r.Post("/api/v1/users/batch", h.User.BatchGet)
		r.Put("/api/v1/users/me", h.User.UpdateMe)
		r.Get("/api/v1/users/search", h.User.Search)
		r.Get("/api/v1/groups/search", h.Conversation.SearchGroups)

		r.Get("/api/v1/conversations", h.Conversation.List)
		r.Get("/api/v1/conversations/{conv_id}", h.Conversation.GetDetail)
		r.Post("/api/v1/conversations/group", h.Conversation.CreateGroup)
		r.Post("/api/v1/conversations/p2p", h.Conversation.CreateP2P)
		r.Put("/api/v1/conversations/{conv_id}", h.Conversation.UpdateGroup)
		r.Post("/api/v1/conversations/{conv_id}/members", h.Conversation.AddMembers)
		r.Delete("/api/v1/conversations/{conv_id}/members/{user_id}", h.Conversation.RemoveMember)
		r.Post("/api/v1/conversations/{conv_id}/leave", h.Conversation.Leave)
		r.Post("/api/v1/conversations/{conv_id}/read", h.Conversation.MarkRead)
		r.Post("/api/v1/conversations/{conv_id}/join-requests", h.Conversation.RequestJoin)
		r.Get("/api/v1/conversations/{conv_id}/join-requests", h.Conversation.ListJoinRequests)
		r.Post("/api/v1/conversations/{conv_id}/join-requests/{user_id}/approve", h.Conversation.ApproveJoinRequest)
		r.Post("/api/v1/conversations/{conv_id}/join-requests/{user_id}/reject", h.Conversation.RejectJoinRequest)
		r.Get("/api/v1/conversations/unread/total", h.Conversation.UnreadTotal)

		r.Get("/api/v1/conversations/{conv_id}/messages", h.Message.GetHistory)

		r.Get("/api/v1/contacts", h.Contact.List)
		r.Post("/api/v1/contacts", h.Contact.Add)
		r.Delete("/api/v1/contacts/{user_id}", h.Contact.Remove)
		r.Put("/api/v1/contacts/{user_id}", h.Contact.UpdateNickname)

		r.Get("/api/v1/sessions", h.Session.ListSessions)
		r.Delete("/api/v1/sessions/{session_id}", h.Session.DeleteSession)

		// File upload (authenticated)
		r.Post("/api/v1/files/upload", h.File.Upload)
		r.Get("/api/v1/files/{file_id}", h.File.GetInfo)
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
		log.Printf("%s %s %d %s",
			r.Method,
			sanitized,
			ww.Status(),
			latency.Round(time.Millisecond),
		)
	})
}

var _ = strings.TrimSpace // ensure "strings" import is used
