package api

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"ziziphus/internal/auth"
	"ziziphus/internal/gateway"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
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
// @Summary List sessions
// @Description List all active sessions for the current user
// @Tags sessions
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {array} APIResponse
// @Router /sessions [get]
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
// @Summary Delete a session
// @Description Revoke a session and force-disconnect the WebSocket
// @Tags sessions
// @Accept json
// @Produce json
// @Security Bearer
// @Param session_id path string true "Session ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /sessions/{session_id} [delete]
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
