package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/dolphinz/im-server/pkg/i18n"
)

type Handlers struct {
	User         *UserHandler
	Conversation *ConvHandler
	Message      *MsgHandler
	Contact      *ContactHandler
}

func NewRouter(h *Handlers, authMW func(http.Handler) http.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(chimw.RealIP)
	r.Use(i18n.Middleware)

	// public routes
	r.Group(func(r chi.Router) {
		r.Post("/api/v1/users/register", h.User.Register)
		r.Post("/api/v1/users/login", h.User.Login)
		r.Get("/metrics", promhttp.Handler().ServeHTTP)
	})

	// authenticated routes
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Get("/api/v1/users/me", h.User.GetMe)
		r.Get("/api/v1/users/{user_id}", h.User.GetUser)
		r.Post("/api/v1/users/batch", h.User.BatchGet)
		r.Put("/api/v1/users/me", h.User.UpdateMe)
		r.Get("/api/v1/users/search", h.User.Search)

		r.Get("/api/v1/conversations", h.Conversation.List)
		r.Get("/api/v1/conversations/{conv_id}", h.Conversation.GetDetail)
		r.Post("/api/v1/conversations/group", h.Conversation.CreateGroup)
		r.Post("/api/v1/conversations/p2p", h.Conversation.CreateP2P)
		r.Put("/api/v1/conversations/{conv_id}", h.Conversation.UpdateGroup)
		r.Post("/api/v1/conversations/{conv_id}/members", h.Conversation.AddMembers)
		r.Delete("/api/v1/conversations/{conv_id}/members/{user_id}", h.Conversation.RemoveMember)
		r.Post("/api/v1/conversations/{conv_id}/leave", h.Conversation.Leave)
		r.Post("/api/v1/conversations/{conv_id}/read", h.Conversation.MarkRead)
		r.Get("/api/v1/conversations/unread/total", h.Conversation.UnreadTotal)

		r.Get("/api/v1/conversations/{conv_id}/messages", h.Message.GetHistory)

		r.Get("/api/v1/contacts", h.Contact.List)
		r.Post("/api/v1/contacts", h.Contact.Add)
		r.Delete("/api/v1/contacts/{user_id}", h.Contact.Remove)
		r.Put("/api/v1/contacts/{user_id}", h.Contact.UpdateNickname)
	})

	return r
}
