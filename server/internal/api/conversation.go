package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/internal/storage/db"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type userGetter interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type convDataRepo interface {
	GetUserConvs(ctx context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error)
	UpdateNameAvatar(ctx context.Context, convID, name, avatar string) error
	UpdateNotice(ctx context.Context, convID, notice string) error
	Pin(ctx context.Context, userID, convID string) error
	Unpin(ctx context.Context, userID, convID string) error
	Clone(ctx context.Context, srcConvID, newConvID, ownerID string, name string, idGen func() int64) error
	SearchByName(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error)
}

type ConvHandler struct {
	convMgr    convManager
	convRepo   convDataRepo
	seqCache   convSeqCache
	readMarker readMarker
	sysMsg     sysMsgSender
	userGetter userGetter
	idGen      func() int64
}

type sysMsgSender interface {
	SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error)
}

type readMarker interface {
	MarkRead(ctx context.Context, userID, convID string, msgID int64) error
}

type convManager interface {
	Get(ctx context.Context, convID string) (*model.Conversation, error)
	CreateGroup(ctx context.Context, name, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error)
	GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error)
	AddMember(ctx context.Context, convID, userID, operatorID string) error
	RemoveMember(ctx context.Context, convID, userID, operatorID string) error
	Leave(ctx context.Context, convID, userID string) error
	GetMembers(ctx context.Context, convID string) ([]*model.ConvMember, error)
	IsMember(ctx context.Context, convID, userID string) (bool, error)
	GetMemberRole(ctx context.Context, convID, userID string) (model.ConvRole, error)
	RequestJoin(ctx context.Context, convID, userID string) error
	ListJoinRequests(ctx context.Context, convID, operatorID string) ([]*model.JoinRequest, error)
	ApproveJoinRequest(ctx context.Context, convID, userID, operatorID string) error
	RejectJoinRequest(ctx context.Context, convID, userID, operatorID string) error
}

type convSeqCache interface {
	GetUnreadCount(ctx context.Context, userID, convID string) (int64, error)
}

func NewConvHandler(convMgr convManager, convRepo convDataRepo, seqCache convSeqCache, readMarker readMarker, sysMsg sysMsgSender, userGetter userGetter, idGen func() int64) *ConvHandler {
	return &ConvHandler{convMgr: convMgr, convRepo: convRepo, seqCache: seqCache, readMarker: readMarker, sysMsg: sysMsg, userGetter: userGetter, idGen: idGen}
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
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
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
		NotFound(w, r)
		return
	}

	isMember, err := h.convMgr.IsMember(r.Context(), convID, userID)
	if err != nil || !isMember {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.not_in_conv_specific")})
		return
	}

	members, err := h.convMgr.GetMembers(r.Context(), convID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	unread, _ := h.seqCache.GetUnreadCount(r.Context(), userID, convID)

	JSON(w, map[string]interface{}{
		"conv_id":      conv.ConvID,
		"type":         conv.Type,
		"name":         conv.Name,
		"owner_id":     conv.OwnerID,
		"avatar":       conv.Avatar,
		"notice":       conv.Notice,
		"members":      members,
		"unread_count": unread,
		"created_at":   conv.CreatedAt,
	})
}

type updateGroupReq struct {
	Name   *string `json:"name"`
	Avatar *string `json:"avatar"`
	Notice *string `json:"notice"`
}

func (h *ConvHandler) UpdateGroup(w http.ResponseWriter, r *http.Request) {
	var req updateGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	convID := chi.URLParam(r, "conv_id")

	conv, err := h.convMgr.Get(r.Context(), convID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if conv.Type != model.ConvGroup {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.group_only")})
		return
	}

	// Only owner or admin can update group info
	userID := auth.UserFromCtx(r.Context())
	role, err := h.convMgr.GetMemberRole(r.Context(), convID, userID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if role < model.ConvRoleAdmin {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	name := conv.Name
	if req.Name != nil {
		name = *req.Name
	}
	avatar := conv.Avatar
	if req.Avatar != nil {
		avatar = *req.Avatar
	}
	// Notice can only be changed by the group owner
	if req.Notice != nil {
		if role != model.ConvRoleOwner {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.owner_only")})
			return
		}
		if err := h.convRepo.UpdateNotice(r.Context(), convID, *req.Notice); err != nil {
			logger.Error("update notice failed", "conv_id", convID, "error", err)
			Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
			return
		}
	}

	if err := h.convRepo.UpdateNameAvatar(r.Context(), convID, name, avatar); err != nil {
		logger.Error("update group failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]interface{}{
		"conv_id": convID,
		"name":    name,
		"avatar":  avatar,
	})
}

func (h *ConvHandler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	var req createGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Name == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.name_required"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	// Deduplicate and filter empty member_ids
	seen := map[string]struct{}{userID: {}}
	var uniqueIDs []string
	for _, mid := range req.MemberIDs {
		if mid != "" && mid != userID {
			if _, ok := seen[mid]; !ok {
				seen[mid] = struct{}{}
				uniqueIDs = append(uniqueIDs, mid)
			}
		}
	}
	req.MemberIDs = uniqueIDs
	conv, err := h.convMgr.CreateGroup(r.Context(), req.Name, userID, req.MemberIDs, h.idGen)
	if err != nil {
		logger.Error("create group failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), conv.ConvID, i18n.T(r.Context(), "sys.group_created", userID, conv.Name), userID)
	}

	JSON(w, map[string]interface{}{
		"conv_id": conv.ConvID,
		"name":    conv.Name,
	})
}

type createP2PReq struct {
	UserID string `json:"user_id"`
}

func (h *ConvHandler) CreateP2P(w http.ResponseWriter, r *http.Request) {
	var req createP2PReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.UserID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.user_id_required"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	if userID == req.UserID {
		BadRequest(w, r, i18n.T(r.Context(), "err.cannot_chat_self"))
		return
	}

	conv, err := h.convMgr.GetOrCreateP2P(r.Context(), userID, req.UserID)
	if err != nil {
		logger.Error("create p2p failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Resolve the partner's display name
	partnerName := ""
	if partner, err := h.userGetter.GetByID(r.Context(), req.UserID); err == nil {
		partnerName = partner.Name
	}

	JSON(w, map[string]interface{}{
		"conv_id": conv.ConvID,
		"type":    conv.Type,
		"name":    partnerName,
	})
}

type addMembersReq struct {
	UserIDs []string `json:"user_ids"`
}

func (h *ConvHandler) AddMembers(w http.ResponseWriter, r *http.Request) {
	var req addMembersReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if len(req.UserIDs) == 0 {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	// Deduplicate user_ids
	seen := make(map[string]struct{}, len(req.UserIDs))
	unique := make([]string, 0, len(req.UserIDs))
	for _, uid := range req.UserIDs {
		if _, ok := seen[uid]; !ok {
			seen[uid] = struct{}{}
			unique = append(unique, uid)
		}
	}
	req.UserIDs = unique

	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	var succeeded []string
	var firstErr error
	for _, mid := range req.UserIDs {
		if err := h.convMgr.AddMember(r.Context(), convID, mid, userID); err != nil {
			logger.Warn("add member failed", "conv_id", convID, "member_id", mid, "error", err)
			if firstErr == nil {
				firstErr = err
			}
		} else {
			succeeded = append(succeeded, mid)
		}
	}
	if h.sysMsg != nil {
		for _, mid := range succeeded {
			h.sysMsg.SendSystemMessage(r.Context(), convID, i18n.T(r.Context(), "sys.member_added", mid), userID)
		}
	}
	if len(succeeded) == 0 && firstErr != nil {
		if appErr, ok := firstErr.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

func (h *ConvHandler) RequestJoin(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.RequestJoin(r.Context(), convID, userID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

func (h *ConvHandler) ListJoinRequests(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	requests, err := h.convMgr.ListJoinRequests(r.Context(), convID, userID)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if requests == nil {
		requests = []*model.JoinRequest{}
	}
	JSON(w, requests)
}

func (h *ConvHandler) ApproveJoinRequest(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := chi.URLParam(r, "user_id")
	operatorID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.ApproveJoinRequest(r.Context(), convID, userID, operatorID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID, "user_id": userID})
}

func (h *ConvHandler) RejectJoinRequest(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := chi.URLParam(r, "user_id")
	operatorID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.RejectJoinRequest(r.Context(), convID, userID, operatorID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID, "user_id": userID})
}

func (h *ConvHandler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	targetID := chi.URLParam(r, "user_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.RemoveMember(r.Context(), convID, targetID, userID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), convID, i18n.T(r.Context(), "sys.member_removed", targetID), userID)
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

func (h *ConvHandler) Leave(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.Leave(r.Context(), convID, userID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sysMsg.SendSystemMessage(r.Context(), convID, i18n.T(r.Context(), "sys.member_left", userID), userID)
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

type markReadReq struct {
	MsgID int64 `json:"msg_id"`
}

func (h *ConvHandler) MarkRead(w http.ResponseWriter, r *http.Request) {
	var req markReadReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	isMember, err := h.convMgr.IsMember(r.Context(), convID, userID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if !isMember {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.not_in_conv_specific")})
		return
	}

	if h.readMarker != nil {
		h.readMarker.MarkRead(r.Context(), userID, convID, req.MsgID)
	}

	JSON(w, map[string]interface{}{"conv_id": convID, "msg_id": req.MsgID})
}

func (h *ConvHandler) SearchGroups(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	items, total, err := h.convRepo.SearchByName(r.Context(), q, page, size)
	if err != nil {
		logger.Error("search groups failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if items == nil {
		items = []*db.GroupSearchItem{}
	}
	Paginated(w, items, total, page, size)
}

func (h *ConvHandler) UnreadTotal(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())

	items, _, err := h.convRepo.GetUserConvs(r.Context(), userID, 1, 1000)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	var totalUnread int64
	for _, item := range items {
		totalUnread += item.UnreadCount
	}
	JSON(w, map[string]interface{}{"total": totalUnread})
}

func (h *ConvHandler) Pin(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")
	if err := h.convRepo.Pin(r.Context(), userID, convID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID, "pinned": true})
}

func (h *ConvHandler) Unpin(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")
	if err := h.convRepo.Unpin(r.Context(), userID, convID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID, "pinned": false})
}

type cloneGroupReq struct {
	Name string `json:"name"`
}

func (h *ConvHandler) Clone(w http.ResponseWriter, r *http.Request) {
	var req cloneGroupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	conv, err := h.convMgr.Get(r.Context(), convID)
	if err != nil || conv.Type != model.ConvGroup {
		NotFound(w, r)
		return
	}

	role, _ := h.convMgr.GetMemberRole(r.Context(), convID, userID)
	if role != model.ConvRoleOwner {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	name := conv.Name + " (副本)"
	if req.Name != "" {
		name = req.Name
	}

	newID := "group_" + strconv.FormatInt(h.idGen(), 10)
	if err := h.convRepo.Clone(r.Context(), convID, newID, userID, name, h.idGen); err != nil {
		logger.Error("clone group failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": newID, "name": name})
}
