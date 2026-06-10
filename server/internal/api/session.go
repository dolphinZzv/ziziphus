package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/gateway"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type sessionManager interface {
	GetUserSessionIDs(ctx context.Context, userID string) []string
	Get(ctx context.Context, sessionID string) *model.Session
	Delete(ctx context.Context, sessionID string) error
}

type SessionHandler struct {
	sessMgr sessionManager
	gwMgr   *gateway.Manager
}

func NewSessionHandler(sessMgr sessionManager, gwMgr *gateway.Manager) *SessionHandler {
	return &SessionHandler{sessMgr: sessMgr, gwMgr: gwMgr}
}

// ListSessions returns all active sessions for the current user.
func (h *SessionHandler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	sessionIDs := h.sessMgr.GetUserSessionIDs(r.Context(), userID)
	sessions := make([]*model.Session, 0, len(sessionIDs))
	for _, sid := range sessionIDs {
		s := h.sessMgr.Get(r.Context(), sid)
		if s != nil {
			sessions = append(sessions, s)
		}
	}
	JSON(w, sessions)
}

// DeleteSession revokes a session: removes it from storage and force-disconnects the WebSocket.
func (h *SessionHandler) DeleteSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	sessionID := chi.URLParam(r, "session_id")
	if sessionID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	s := h.sessMgr.Get(r.Context(), sessionID)
	if s == nil || s.UserID != userID {
		Error(w, r, http.StatusNotFound, &model.AppError{Code: model.ErrNotFound, Message: i18n.T(r.Context(), "err.resource_not_found")})
		return
	}

	// Force-disconnect WebSocket if connected
	h.gwMgr.DisconnectBySessionID(r.Context(), sessionID)

	// Remove from storage
	if err := h.sessMgr.Delete(r.Context(), sessionID); err != nil {
		logger.Error("delete session failed", "session_id", sessionID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	logger.Info("session revoked", "session_id", sessionID, "user_id", userID)
	JSON(w, map[string]string{"session_id": sessionID})
}
