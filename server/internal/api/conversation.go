package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"ziziphus/internal/auth"
	"ziziphus/internal/storage/db"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

type userGetter interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
}

type convDataRepo interface {
	GetUserConvs(ctx context.Context, userID string, page, size int) ([]*db.ConvListItem, int, error)
	UpdateNameAvatar(ctx context.Context, convID, name, avatar string) error
	UpdateNotice(ctx context.Context, convID, notice string) error
	UpdateCover(ctx context.Context, convID, cover string) error
	UpdatePrimaryColor(ctx context.Context, convID, color string) error
	Pin(ctx context.Context, userID, convID string) error
	Unpin(ctx context.Context, userID, convID string) error
	Clone(ctx context.Context, srcConvID, newConvID, ownerID string, name string, idGen func() int64) error
	SearchByName(ctx context.Context, q string, page, size int) ([]*db.GroupSearchItem, int, error)
	AreContacts(ctx context.Context, userA, userB string) (bool, error)
	UpdateSettings(ctx context.Context, convID string, settings map[string]any) error
	GetSettings(ctx context.Context, convID string) (map[string]any, error)
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
	CreateGroup(ctx context.Context, name, headline, ownerID string, memberIDs []string, idGen func() int64) (*model.Conversation, error)
	GetOrCreateP2P(ctx context.Context, userA, userB string) (*model.Conversation, error)
	AddMember(ctx context.Context, convID, userID, operatorID string) error
	RemoveMember(ctx context.Context, convID, userID, operatorID string) error
	Leave(ctx context.Context, convID, userID string) error
	Disband(ctx context.Context, convID, ownerID string) error
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

// sendSysMsgWithName resolves a user name and sends a formatted system message.
func (h *ConvHandler) sendSysMsgWithName(ctx context.Context, convID, key, userID string, extra ...string) {
	name := userID
	if u, err := h.userGetter.GetByID(ctx, userID); err == nil && u.Name != "" {
		name = u.Name
	}
	args := []interface{}{name}
	for _, e := range extra {
		args = append(args, e)
	}
	body := i18n.T(ctx, key, args...)
	_, _ = h.sysMsg.SendSystemMessage(ctx, convID, body, userID)
}

type createGroupReq struct {
	Name         string   `json:"name"`
	MemberIDs    []string `json:"member_ids"`
	Headline     string   `json:"headline"`
	PrimaryColor string   `json:"primary_color,omitempty"`
}

// @Summary List conversations
// @Description Get paginated list of conversations for the current user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(20)
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations [get]
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

// @Summary Get conversation detail
// @Description Get detailed information about a conversation including members
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id} [get]
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
		"conv_id":       conv.ConvID,
		"type":          conv.Type,
		"name":          conv.Name,
		"owner_id":      conv.OwnerID,
		"avatar":        conv.Avatar,
		"cover":         conv.Cover,
		"notice":        conv.Notice,
		"headline":      conv.Headline,
		"primary_color": conv.PrimaryColor,
		"members":       members,
		"unread_count":  unread,
		"created_at":    conv.CreatedAt,
	})
}

type updateGroupReq struct {
	Name         *string `json:"name"`
	Avatar       *string `json:"avatar"`
	Notice       *string `json:"notice"`
	Cover        *string `json:"cover"`
	PrimaryColor *string `json:"primary_color"`
}

// @Summary Update group conversation
// @Description Update group name, avatar, notice, or cover. Name/avatar require admin+, notice requires owner, cover supports any member.
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param body body updateGroupReq true "Update group request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id} [put]
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

	userID := auth.UserFromCtx(r.Context())
	isMember, err := h.convMgr.IsMember(r.Context(), convID, userID)
	if err != nil || !isMember {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
		return
	}

	isGroup := conv.Type == model.ConvGroup

	// Name and avatar can only be updated for groups
	if (req.Name != nil || req.Avatar != nil) && !isGroup {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.group_only")})
		return
	}

	// For groups, admin+ permission is required for name/avatar/cover updates
	// For P2P, any member can update cover
	var role model.ConvRole
	if isGroup {
		role, err = h.convMgr.GetMemberRole(r.Context(), convID, userID)
		if err != nil {
			NotFound(w, r)
			return
		}
	}

	// Name update (group, admin+)
	if req.Name != nil {
		if role < model.ConvRoleAdmin {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
			return
		}
	}
	// Avatar update (group, admin+)
	if req.Avatar != nil {
		if role < model.ConvRoleAdmin {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
			return
		}
	}
	// Notice update (group, owner only)
	if req.Notice != nil {
		if !isGroup {
			Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.group_only")})
			return
		}
		if role != model.ConvRoleOwner {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.owner_only")})
			return
		}
	}
	// Cover update (group: admin+, P2P: any member)
	if req.Cover != nil {
		if isGroup && role < model.ConvRoleAdmin {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.permission_denied")})
			return
		}
	}

	name := conv.Name
	if req.Name != nil {
		name = *req.Name
	}
	avatar := conv.Avatar
	if req.Avatar != nil {
		avatar = *req.Avatar
	}

	// Apply name/avatar update
	if req.Name != nil || req.Avatar != nil {
		if err := h.convRepo.UpdateNameAvatar(r.Context(), convID, name, avatar); err != nil {
			logger.Error("update group failed", "conv_id", convID, "error", err)
			Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
			return
		}
	}

	// Apply notice update
	if req.Notice != nil {
		if err := h.convRepo.UpdateNotice(r.Context(), convID, *req.Notice); err != nil {
			logger.Error("update notice failed", "conv_id", convID, "error", err)
			Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
			return
		}
	}

	// Apply cover update
	if req.Cover != nil {
		if err := h.convRepo.UpdateCover(r.Context(), convID, *req.Cover); err != nil {
			logger.Error("update cover failed", "conv_id", convID, "error", err)
			Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
			return
		}
	}

	// Apply primary_color update
	if req.PrimaryColor != nil {
		if err := h.convRepo.UpdatePrimaryColor(r.Context(), convID, *req.PrimaryColor); err != nil {
			logger.Error("update primary_color failed", "conv_id", convID, "error", err)
			Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
			return
		}
	}

	resp := map[string]interface{}{
		"conv_id": convID,
		"name":    name,
		"avatar":  avatar,
	}
	if req.Cover != nil {
		resp["cover"] = *req.Cover
	} else {
		resp["cover"] = conv.Cover
	}
	if req.PrimaryColor != nil {
		resp["primary_color"] = *req.PrimaryColor
	}
	JSON(w, resp)
}

// @Summary Create a group conversation
// @Description Create a new group conversation with specified members
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body createGroupReq true "Create group request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/group [post]
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
	conv, err := h.convMgr.CreateGroup(r.Context(), req.Name, req.Headline, userID, req.MemberIDs, h.idGen)
	if err != nil {
		logger.Error("create group failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sendSysMsgWithName(r.Context(), conv.ConvID, "sys.group_created", userID, conv.Name)
	}

	if req.PrimaryColor != "" {
		_ = h.convRepo.UpdatePrimaryColor(r.Context(), conv.ConvID, req.PrimaryColor)
	}

	JSON(w, map[string]interface{}{
		"conv_id": conv.ConvID,
		"name":    conv.Name,
	})
}

type createP2PReq struct {
	UserID string `json:"user_id"`
}

// @Summary Create or get a P2P conversation
// @Description Create or get an existing one-on-one conversation with another user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param body body createP2PReq true "Create P2P request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/p2p [post]
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

	// Check if target user allows direct chat
	partner, partnerErr := h.userGetter.GetByID(r.Context(), req.UserID)
	if partnerErr == nil && !partner.AllowDirectChat && partner.Type == model.UserHuman {
		// Allow contacts to bypass this restriction
		areContacts, _ := h.convRepo.AreContacts(r.Context(), userID, req.UserID)
		if !areContacts {
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.direct_chat_disabled")})
			return
		}
	}

	conv, err := h.convMgr.GetOrCreateP2P(r.Context(), userID, req.UserID)
	if err != nil {
		logger.Error("create p2p failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Resolve the partner's display name
	partnerName := ""
	if partnerErr == nil {
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

// @Summary Add members to a conversation
// @Description Add one or more members to a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param body body addMembersReq true "Add members request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/members [post]
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
			h.sendSysMsgWithName(r.Context(), convID, "sys.member_added", mid)
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

// @Summary Request to join a conversation
// @Description Submit a join request for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/join-requests [post]
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

// @Summary List join requests
// @Description List pending join requests for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {array} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/join-requests [get]
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

// @Summary Approve a join request
// @Description Approve a pending join request for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/join-requests/{user_id}/approve [post]
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

// @Summary Reject a join request
// @Description Reject a pending join request for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param user_id path string true "User ID"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/join-requests/{user_id}/reject [post]
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

// @Summary Remove a member from a conversation
// @Description Remove a specified member from a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param user_id path string true "User ID of the member to remove"
// @Success 200 {object} map[string]interface{}
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/members/{user_id} [delete]
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
		h.sendSysMsgWithName(r.Context(), convID, "sys.member_removed", targetID)
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

// @Summary Leave a conversation
// @Description Leave a conversation as the current user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/leave [post]
func (h *ConvHandler) Leave(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.Leave(r.Context(), convID, userID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if h.sysMsg != nil {
		h.sendSysMsgWithName(r.Context(), convID, "sys.member_left", userID)
	}
	JSON(w, map[string]interface{}{"conv_id": convID})
}

type markReadReq struct {
	MsgID int64 `json:"msg_id"`
}

// @Summary Mark conversation as read
// @Description Mark messages up to a specified message ID as read
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param body body markReadReq true "Mark read request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/read [post]
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
		_ = h.readMarker.MarkRead(r.Context(), userID, convID, req.MsgID)
	}

	JSON(w, map[string]interface{}{"conv_id": convID, "msg_id": req.MsgID})
}

// @Summary Search groups
// @Description Search groups by name with pagination
// @Tags groups
// @Accept json
// @Produce json
// @Security Bearer
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param size query int false "Page size" default(20)
// @Success 200 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /groups/search [get]
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

// @Summary Get total unread count
// @Description Get the total number of unread messages across all conversations for the current user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIResponse
// @Router /conversations/unread/total [get]
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

// @Summary Pin a conversation
// @Description Pin a conversation for the current user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/pin [post]
func (h *ConvHandler) Pin(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	convID := chi.URLParam(r, "conv_id")
	if err := h.convRepo.Pin(r.Context(), userID, convID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"conv_id": convID, "pinned": true})
}

// @Summary Unpin a conversation
// @Description Unpin a conversation for the current user
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} map[string]interface{}
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/unpin [post]
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

// @Summary Get conversation settings
// @Description Get settings for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Router /conversations/{conv_id}/settings [get]
func (h *ConvHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if ok, _ := h.convMgr.IsMember(r.Context(), convID, userID); !ok {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.not_in_conv")})
		return
	}

	settings, err := h.convRepo.GetSettings(r.Context(), convID)
	if err != nil {
		settings = map[string]any{}
	}
	JSON(w, map[string]any{"settings": settings})
}

// @Summary Update conversation settings
// @Description Update settings for a conversation
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/settings [put]
func (h *ConvHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if ok, _ := h.convMgr.IsMember(r.Context(), convID, userID); !ok {
		Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: i18n.T(r.Context(), "err.not_in_conv")})
		return
	}

	var body struct {
		Settings map[string]any `json:"settings"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_json"))
		return
	}

	if err := h.convRepo.UpdateSettings(r.Context(), convID, body.Settings); err != nil {
		logger.Error("update settings failed", "conv_id", convID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]any{"settings": body.Settings})
}

// @Summary Disband a group conversation
// @Description Disband a group conversation (owner only)
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Success 200 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/disband [post]
func (h *ConvHandler) Disband(w http.ResponseWriter, r *http.Request) {
	convID := chi.URLParam(r, "conv_id")
	userID := auth.UserFromCtx(r.Context())

	if err := h.convMgr.Disband(r.Context(), convID, userID); err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusForbidden, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "disbanded"})
}

// @Summary Clone a group conversation
// @Description Clone a group conversation with all members and settings (owner only)
// @Tags conversations
// @Accept json
// @Produce json
// @Security Bearer
// @Param conv_id path string true "Conversation ID"
// @Param body body cloneGroupReq false "Clone group request"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} APIResponse
// @Failure 403 {object} APIResponse
// @Failure 404 {object} APIResponse
// @Failure 500 {object} APIResponse
// @Router /conversations/{conv_id}/clone [post]
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

	name := conv.Name + " (copy)"
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
