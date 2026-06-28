package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"siciv.space/agent/panda_ai/internal/auth"
	"siciv.space/agent/panda_ai/pkg/i18n"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type contactStorage interface {
	Add(ctx context.Context, c *model.Contact) error
	Remove(ctx context.Context, userID, contactID string) error
	List(ctx context.Context, userID string, page, size int) ([]*model.Contact, int, error)
	UpdateNickname(ctx context.Context, userID, contactID, nickname string) error
}

type contactRequestStorage interface {
	Insert(ctx context.Context, req *model.ContactRequest) (int64, error)
	UpdateFormMsgID(ctx context.Context, id, formMsgID int64) error
	GetByID(ctx context.Context, id int64) (*model.ContactRequest, error)
	GetByFormMsgID(ctx context.Context, formMsgID int64) (*model.ContactRequest, error)
	GetByPair(ctx context.Context, fromUserID, toUserID string) (*model.ContactRequest, error)
	ListSent(ctx context.Context, userID string, page, size int) ([]*model.ContactRequest, int, error)
	ListReceived(ctx context.Context, userID string, status int, page, size int) ([]*model.ContactRequest, int, error)
	Delete(ctx context.Context, id int64) error
	ExistsAnyDirection(ctx context.Context, userA, userB string) (bool, error)
}

type formMessageSender interface {
	SendFormMessage(ctx context.Context, convID string, body *model.FormDefinitionBody) (*model.Message, error)
	SendSystemMessage(ctx context.Context, convID, body string, senderID ...string) (*model.Message, error)
}

type systemConvManager interface {
	GetOrCreateSystemConv(ctx context.Context, userID string) (*model.Conversation, error)
	Leave(ctx context.Context, convID, userID string) error
}

type userQueryRepo interface {
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
}

type ContactHandler struct {
	contactRepo contactStorage
	reqRepo     contactRequestStorage
	userRepo    userQueryRepo
	sessMgr     sessionChecker
	ingest      formMessageSender
	convMgr     systemConvManager
}

func NewContactHandler(contactRepo contactStorage, reqRepo contactRequestStorage, userRepo userQueryRepo, sessMgr sessionChecker, ingest formMessageSender, convMgr systemConvManager) *ContactHandler {
	return &ContactHandler{
		contactRepo: contactRepo,
		reqRepo:     reqRepo,
		userRepo:    userRepo,
		sessMgr:     sessMgr,
		ingest:      ingest,
		convMgr:     convMgr,
	}
}

// ---------------------------------------------------------------------------
// Existing endpoints
// ---------------------------------------------------------------------------

type addContactReq struct {
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname"`
}

func (h *ContactHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	contacts, total, err := h.contactRepo.List(r.Context(), userID, page, size)
	if err != nil {
		logger.Error("list contacts failed", "user_id", userID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// enrich with user info
	contactIDs := make([]string, len(contacts))
	for i, c := range contacts {
		contactIDs[i] = c.ContactID
	}
	userMap, _ := h.userRepo.GetByIDs(r.Context(), contactIDs)

	items := make([]map[string]interface{}, 0, len(contacts))
	for _, c := range contacts {
		item := map[string]interface{}{
			"user_id":  c.ContactID,
			"nickname": c.Nickname,
			"added_at": c.AddedAt,
		}
		if u, ok := userMap[c.ContactID]; ok {
			item["name"] = u.Name
			item["avatar"] = u.Avatar
			if h.sessMgr.IsOnline(r.Context(), u.ID) {
				item["status"] = model.UserOnline
			} else {
				item["status"] = model.UserOffline
			}
		}
		items = append(items, item)
	}
	Paginated(w, items, total, page, size)
}

func (h *ContactHandler) Add(w http.ResponseWriter, r *http.Request) {
	var req addContactReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.UserID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.contact_user_id_required"))
		return
	}

	userID := auth.UserFromCtx(r.Context())
	if req.UserID == userID {
		BadRequest(w, r, i18n.T(r.Context(), "err.cannot_add_self"))
		return
	}

	contact := &model.Contact{
		UserID:    userID,
		ContactID: req.UserID,
		Nickname:  req.Nickname,
		AddedAt:   time.Now().UnixMilli(),
	}
	if err := h.contactRepo.Add(r.Context(), contact); err != nil {
		logger.Error("add contact failed", "user_id", userID, "contact_id", req.UserID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"user_id": req.UserID, "nickname": req.Nickname})
}

// Remove now performs bidirectional deletion and removes the P2P conversation.
func (h *ContactHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	contactID := chi.URLParam(r, "user_id")

	if err := h.contactRepo.Remove(r.Context(), userID, contactID); err != nil {
		logger.Error("remove contact failed", "user_id", userID, "contact_id", contactID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	// Also remove the reverse direction so the relationship is fully severed.
	if err := h.contactRepo.Remove(r.Context(), contactID, userID); err != nil {
		logger.Error("remove reverse contact failed", "user_id", contactID, "contact_id", userID, "error", err)
	}

	// Remove the P2P conversation if it exists.
	if h.convMgr != nil {
		p2pConvID := model.MakeP2PConvID(userID, contactID)
		h.convMgr.Leave(r.Context(), p2pConvID, userID)
		h.convMgr.Leave(r.Context(), p2pConvID, contactID)
	}

	JSON(w, map[string]interface{}{"user_id": contactID})
}

type updateContactNicknameReq struct {
	Nickname string `json:"nickname"`
}

func (h *ContactHandler) UpdateNickname(w http.ResponseWriter, r *http.Request) {
	var req updateContactNicknameReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	contactID := chi.URLParam(r, "user_id")

	if err := h.contactRepo.UpdateNickname(r.Context(), userID, contactID, req.Nickname); err != nil {
		logger.Error("update contact nickname failed", "user_id", userID, "contact_id", contactID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"user_id": contactID, "nickname": req.Nickname})
}

// ---------------------------------------------------------------------------
// Friend request endpoints
// ---------------------------------------------------------------------------

type requestContactReq struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

// RequestContact handles POST /api/v1/contact-requests.
// A sends a friend request to B.
func (h *ContactHandler) RequestContact(w http.ResponseWriter, r *http.Request) {
	var req requestContactReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	ctx := r.Context()
	senderID := auth.UserFromCtx(ctx)
	targetID := req.UserID

	// Validate
	if targetID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.contact_user_id_required"))
		return
	}
	if targetID == senderID {
		Error(w, r, http.StatusBadRequest, model.ErrContactRequestSelf)
		return
	}

	// Check target exists
	targetUser, err := h.userRepo.GetByIDs(ctx, []string{targetID})
	if err != nil || len(targetUser) == 0 || targetUser[targetID] == nil {
		Error(w, r, http.StatusNotFound, &model.AppError{Code: model.ErrNotFound, Message: "用户不存在", Key: "err.user_not_found"})
		return
	}
	targetName := targetUser[targetID].Name

	// Check they are not already friends (bidirectional check)
	areFriends, err := h.reqRepo.ExistsAnyDirection(ctx, senderID, targetID)
	if err != nil {
		logger.Error("check existing friendship failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if areFriends {
		Error(w, r, http.StatusConflict, model.ErrAlreadyFriends)
		return
	}

	// Check for existing request (rejected or approved -> allow re-request; pending -> deny)
	existing, err := h.reqRepo.GetByPair(ctx, senderID, targetID)
	if err != nil {
		logger.Error("check existing request failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if existing != nil {
		if existing.Status == model.ContactRequestPending {
			Error(w, r, http.StatusConflict, model.ErrContactRequestDuplicate)
			return
		}
		// Delete old rejected/approved request to make room for the new one.
		h.reqRepo.Delete(ctx, existing.ID)
	}

	// Get sender info for the form
	senderUsers, _ := h.userRepo.GetByIDs(ctx, []string{senderID})
	senderName := senderID
	senderAvatar := ""
	if u, ok := senderUsers[senderID]; ok && u != nil {
		senderName = u.Name
		senderAvatar = u.Avatar
	}

	// Ensure B's system conversation exists (lazy create)
	if _, err := h.convMgr.GetOrCreateSystemConv(ctx, targetID); err != nil {
		logger.Error("ensure system conv for target failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Build form definition
	now := time.Now().UnixMilli()
	formBody := &model.FormDefinitionBody{
		FormID:         uuid.New().String(),
		Type:           model.FormTypeContactRequest,
		Title:          i18n.T(ctx, "contact_request.title"),
		FromUserID:     senderID,
		FromUserName:   senderName,
		FromUserAvatar: senderAvatar,
		RequestID:      0, // will be set after insert
		Message:        req.Message,
		Actions: []model.FormAction{
			{Action: "approve", Label: i18n.T(ctx, "contact_request.approve"), Style: model.FormActionStylePrimary},
			{Action: "reject", Label: i18n.T(ctx, "contact_request.reject"), Style: model.FormActionStyleDanger},
		},
		Status:    model.FormStatusActive,
		CreatedAt: now,
	}

	// Insert contact request first to get the ID
	cr := &model.ContactRequest{
		FromUserID: senderID,
		ToUserID:   targetID,
		Status:     model.ContactRequestPending,
		Message:    req.Message,
	}
	requestID, err := h.reqRepo.Insert(ctx, cr)
	if err != nil {
		logger.Error("insert contact request failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	formBody.RequestID = requestID

	// Send form message to B's system conversation
	formMsg, err := h.ingest.SendFormMessage(ctx, model.MakeSystemConvID(targetID), formBody)
	if err != nil {
		logger.Error("send form message failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	// Update the contact request with the form message ID
	h.reqRepo.UpdateFormMsgID(ctx, requestID, formMsg.MsgID)

	// Ensure A's system conversation exists (lazy create) and send confirmation
	if _, err := h.convMgr.GetOrCreateSystemConv(ctx, senderID); err != nil {
		logger.Error("ensure sender system conv failed", "error", err)
	} else {
		h.ingest.SendSystemMessage(ctx, model.MakeSystemConvID(senderID),
			i18n.T(ctx, "contact_request.sent", targetName))
	}

	JSON(w, map[string]interface{}{
		"request_id":  requestID,
		"form_msg_id": formMsg.MsgID,
	})
}

// ListSentRequests handles GET /api/v1/contact-requests/sent.
func (h *ContactHandler) ListSentRequests(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	requests, total, err := h.reqRepo.ListSent(r.Context(), userID, page, size)
	if err != nil {
		logger.Error("list sent requests failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if requests == nil {
		requests = []*model.ContactRequest{}
	}
	Paginated(w, requests, total, page, size)
}

// ListReceivedRequests handles GET /api/v1/contact-requests/received.
func (h *ContactHandler) ListReceivedRequests(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	statusStr := r.URL.Query().Get("status")
	status := -1 // all
	if statusStr != "" {
		if s, err := strconv.Atoi(statusStr); err == nil {
			status = s
		}
	}
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	requests, total, err := h.reqRepo.ListReceived(r.Context(), userID, status, page, size)
	if err != nil {
		logger.Error("list received requests failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if requests == nil {
		requests = []*model.ContactRequest{}
	}
	Paginated(w, requests, total, page, size)
}

// GetRequestByFormMsgID handles GET /api/v1/contact-requests/by-form/{msg_id}.
func (h *ContactHandler) GetRequestByFormMsgID(w http.ResponseWriter, r *http.Request) {
	msgIDStr := chi.URLParam(r, "msg_id")
	msgID, err := strconv.ParseInt(msgIDStr, 10, 64)
	if err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	req, err := h.reqRepo.GetByFormMsgID(r.Context(), msgID)
	if err != nil {
		logger.Error("get request by form msg_id failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if req == nil {
		Error(w, r, http.StatusNotFound, model.ErrContactRequestNotFound)
		return
	}
	JSON(w, req)
}
