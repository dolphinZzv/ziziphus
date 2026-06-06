package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/dolphinz/im-server/internal/auth"
	"github.com/dolphinz/im-server/internal/storage/db"
	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
)

type ConvHandler struct {
	convMgr     convManager
	convRepo    *db.ConvRepo
	seqCache    convSeqCache
	readMarker  readMarker
	sysMsg      sysMsgSender
	idGen       func() int64
}

type sysMsgSender interface {
	SendSystemMessage(ctx context.Context, convID, body string) (*model.Message, error)
}

type readMarker interface {
	MarkRead(ctx context.Context, userID, convID string, msgID int64) error
}

type convManager interface {
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	CreateGroup(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error)
	AddMember(ctx context.Context, convID, userID, operatorID string) error
	RemoveMember(ctx context.Context, convID, userID, operatorID string) error
	Leave(ctx context.Context, convID, userID string) error
	GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
}

type convSeqCache interface {
	GetUnreadCount(ctx context.Context, userID, convID string) (int64, error)
}

func NewConvHandler(convMgr convManager, convRepo *db.ConvRepo, seqCache convSeqCache, readMarker readMarker, sysMsg sysMsgSender, idGen func() int64) *ConvHandler {
	return &ConvHandler{convMgr: convMgr, convRepo: convRepo, seqCache: seqCache, readMarker: readMarker, sysMsg: sysMsg, idGen: idGen}
}

type createGroupReq struct {
	Name      string   `json:"name"`
	MemberIDs []string `json:"member_ids"`
}

func (h *ConvHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := h.convRepo.GetUserConvs(r.Context(), userID, page, size)
	if err != nil {
		logger.Error("list conversations failed", "user_id", userID, "error", err)
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	for _, item := range items {
		count, _ := h.seqCache.GetUnreadCount(r.Context(), userID, item.ConvID)
		item.UnreadCount = count
	}
	Paginated(w, items, total, page, size)
}

func (h *ConvHandler) GetDetail(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")

	conv, err := h.convMgr.Get(r.Context(), convID)
	if err != nil {
		NotFound(w)
		return
	}

	isMember, err := h.convMgr.IsMember(r.Context(), convID, userID)
	if err != nil || !isMember {
		Error(w, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: "不在会话中"})
		return
	}

	members, err := h.convMgr.GetMembers(r.Context(), convID)
	if err != nil {
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	unread, _ := h.seqCache.GetUnreadCount(r.Context(), userID, convID)

	JSON(w, map[string]interface{}{
		"conv_id":      conv.ConvID,
		"type":         conv.Type,
		"name":         conv.Name,
		"owner_id":     conv.OwnerID,
		"avatar":       conv.Avatar,
		"members":      members,
		"unread_count": unread,
		"created_at":   conv.CreatedAt,
	})
}

type updateGroupReq struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (h *ConvHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	var req updateGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	convID := chi.URLParam(r, "conv_id")

	conv, err := h.convMgr.Get(r.Context(), convID)
	if err != nil {
		NotFound(w)
		return
	}
	if conv.Type != model.ConvGroup {
		Error(w, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "仅支持群组"})
		return
	}

	if err := h.convRepo.UpdateNameAvatar(r.Context(), convID, req.Name, req.Avatar); err != nil {
		logger.Error("update group failed", "conv_id", convID, "error", err)
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]interface{}{
		"conv_id": convID,
		"name":    req.Name,
		"avatar":  req.Avatar,
	})
}

func (h *ConvHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	if req.Name == "" {
		BadRequest(w, "群组名称不能为空")
		return
	}
	userID := auth.UserFromCtx(r.Context())
	conv, err := h.convMgr.CreateGroup(r.Context(), req.Name, userID, req.MemberIDs, h.idGen)
	if err != nil {
		logger.Error("create group failed", "error", err)
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), conv.ConvID, userID+" 创建了群")
	}

	JSON(w, map[string]interface{}{
		"conv_id": conv.ConvID,
		"name":    conv.Name,
	})
}

type addMembersReq struct {
	UserIDs []string `json:"user_ids"`
}

func (h *ConvHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	var req addMembersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	var lastErr error
	for _, mid := range req.UserIDs {
		if err := h.convMgr.AddMember(r.Context(), convID, mid, userID); err != nil {
			lastErr = err
			logger.Warn("add member failed", "conv_id", convID, "member_id", mid, "error", err)
		}
	}
	if lastErr != nil {
		if appErr, ok := lastErr.(*model.AppError); ok {
			Error(w, http.StatusForbidden, appErr)
			return
		}
	}
	if h.sysMsg != nil {
		for _, mid := range req.UserIDs {
			h.sysMsg.SendSystemMessage(r.Context(), convID, mid+" 被加入群")
		}
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

func (h *ConvHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	targetID := chi.URLParam(r, "user_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.RemoveMember(r.Context(), convID, targetID, userID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, http.StatusForbidden, appErr)
			return
		}
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), convID, targetID+" 被移出群")
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

func (h *ConvHandler) Leave(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.Leave(r.Context(), convID, userID); err != nil {
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), convID, userID+" 退出了群")
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

type markReadReq struct {
	MsgID int64 `json:"msg_id"`
}

func (h *ConvHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "参数错误")
		return
	}
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if h.readMarker != nil {
		h.readMarker.MarkRead(r.Context(), userID, convID, req.MsgID)
	}

	JSON(w, map[string]interface{}{"conv_id": convID, "msg_id": req.MsgID})
}

func (h *ConvHandler) UnreadTotal(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())

	items, _, err := h.convRepo.GetUserConvs(r.Context(), userID, 1, 1000)
	if err != nil {
		Error(w, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	var totalUnread int64
	for _, item := range items {
		totalUnread += item.UnreadCount
	}
	JSON(w, map[string]interface{}{"total": totalUnread})
}
