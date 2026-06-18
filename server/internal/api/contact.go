package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
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

type userQueryRepo interface {
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
}

type ContactHandler struct {
	contactRepo contactStorage
	userRepo    userQueryRepo
	sessMgr     sessionChecker
}

func NewContactHandler(contactRepo contactStorage, userRepo userQueryRepo, sessMgr sessionChecker) *ContactHandler {
	return &ContactHandler{contactRepo: contactRepo, userRepo: userRepo, sessMgr: sessMgr}
}

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

func (h *ContactHandler) Remove(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	contactID := chi.URLParam(r, "user_id")

	if err := h.contactRepo.Remove(r.Context(), userID, contactID); err != nil {
		logger.Error("remove contact failed", "user_id", userID, "contact_id", contactID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
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
