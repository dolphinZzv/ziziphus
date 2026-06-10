package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type convMemberChecker interface {
	IsMember(ctx context.Context, convID, userID string) (bool, error)
}

type msgStorage interface {
	GetHistory(ctx context.Context, convID string, beforeMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error)
}

type MsgHandler struct {
	msgRepo msgStorage
	convMgr convMemberChecker
}

func NewMsgHandler(msgRepo msgStorage, convMgr convMemberChecker) *MsgHandler {
	return &MsgHandler{msgRepo: msgRepo, convMgr: convMgr}
}

func (h *MsgHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	// Verify the user is a member of this conversation
	isMember, err := h.convMgr.IsMember(r.Context(), convID, userID)
	if err != nil || !isMember {
		Error(w, r, http.StatusForbidden, model.ErrConvNotFound)
		return
	}

	before, _ := strconv.ParseInt(r.URL.Query().Get("before_msg_id"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}
	keyword := r.URL.Query().Get("keyword")
	startDate, _ := strconv.ParseInt(r.URL.Query().Get("start_date"), 10, 64)
	endDate, _ := strconv.ParseInt(r.URL.Query().Get("end_date"), 10, 64)

	messages, err := h.msgRepo.GetHistory(r.Context(), convID, before, limit, keyword, startDate, endDate)
	if err != nil {
		logger.Error("get history failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	items := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		items = append(items, map[string]interface{}{
			"msg_id":       m.MsgID,
			"conv_id":      m.ConvID,
			"sender_id":    m.SenderID,
			"sender_name":  m.SenderName,
			"content_type": m.ContentType,
			"body":         m.Body,
			"mention":      m.Mention,
			"reply_to":     m.ReplyTo,
			"timestamp":    m.Timestamp,
			"conv_seq":     m.ConvSeq,
			"status":       m.Status,
		})
	}
	JSON(w, items)
}
