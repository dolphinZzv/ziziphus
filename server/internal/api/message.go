package api

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
)

type msgStorage interface {
	GetHistory(ctx context.Context, convID string, beforeMsgID int64, limit int) ([]*model.Message, error)
}

type MsgHandler struct {
	msgRepo msgStorage
}

func NewMsgHandler(msgRepo msgStorage) *MsgHandler {
	return &MsgHandler{msgRepo: msgRepo}
}

func (h *MsgHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	before, _ := strconv.ParseInt(r.URL.Query().Get("before_msg_id"), 10, 64)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 50
	}

	messages, err := h.msgRepo.GetHistory(r.Context(), convID, before, limit)
	if err != nil {
		logger.Error("get history failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	items := make([]map[string]interface{}, 0, len(messages))
	for _, m := range messages {
		items = append(items, map[string]interface{}{
			"msg_id":     m.MsgID,
			"conv_id":    m.ConvID,
			"sender_id":  m.SenderID,
			"content_type": m.ContentType,
			"body":       m.Body,
			"mention":    m.Mention,
			"reply_to":   m.ReplyTo,
			"timestamp":  m.Timestamp,
			"conv_seq":   m.ConvSeq,
			"status":     m.Status,
		})
	}
	JSON(w, items)
}
