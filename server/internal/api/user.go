package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/dolphinz/im-server/internal/auth"
	"github.com/dolphinz/im-server/pkg/i18n"
	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	authSvc  *auth.Service
	userRepo userRepo
	sessMgr  sessionChecker
}

type userRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
	Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error)
	Update(ctx context.Context, id, name, avatar string) error
}

type sessionChecker interface {
	IsOnline(ctx context.Context, userID string) bool
	GetUserSessionIDs(ctx context.Context, userID string) []string
}

func NewUserHandler(authSvc *auth.Service, userRepo userRepo, sessMgr sessionChecker) *UserHandler {
	return &UserHandler{authSvc: authSvc, userRepo: userRepo, sessMgr: sessMgr}
}

type registerReq struct {
	Name     string `json:"name"`
	Account  string `json:"account"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Name == "" || req.Password == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.name_password_required"))
		return
	}
	user, token, err := h.authSvc.Register(r.Context(), req.Name, req.Password, req.Account)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r,http.StatusBadRequest, appErr)
			return
		}
		logger.Error("register failed", "error", err)
		Error(w, r,http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"user_id": user.ID,
		"account": user.Account,
		"name":    user.Name,
		"token":   token,
	})
}

type loginReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

func (h *UserHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	token, expiresAt, userID, err := h.authSvc.Login(r.Context(), req.Account, req.Password)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r,http.StatusUnauthorized, appErr)
			return
		}
		Error(w, r,http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	user, _ := h.userRepo.GetByID(r.Context(), userID)
	if user != nil {
		user.Password = ""
	}
	JSON(w, map[string]interface{}{
		"user_id":    userID,
		"account":    req.Account,
		"name":       user.Name,
		"token":      token,
		"expires_at": expiresAt,
	})
}

func (h *UserHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		NotFound(w, r)
		return
	}
	user.Password = ""
	writeUserWithDevices(w, r, user, h.sessMgr)
}

func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	if userID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.user_id_required"))
		return
	}
	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		NotFound(w, r)
		return
	}
	user.Password = ""
	writeUserWithDevices(w, r, user, h.sessMgr)
}

func writeUserWithDevices(w http.ResponseWriter, r *http.Request, user *model.User, sessMgr sessionChecker) {
	isOnline := sessMgr.IsOnline(r.Context(), user.ID)
	if isOnline {
		user.Status = model.UserOnline
	} else {
		user.Status = model.UserOffline
	}
	JSON(w, user)
}

type batchReq struct {
	UserIDs []string `json:"user_ids"`
}

func (h *UserHandler) BatchGet(w http.ResponseWriter, r *http.Request) {
	var req batchReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	users, err := h.userRepo.GetByIDs(r.Context(), req.UserIDs)
	if err != nil {
		Error(w, r,http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	result := make(map[string]interface{}, len(users))
	for id, u := range users {
		u.Password = ""
		if h.sessMgr.IsOnline(r.Context(), u.ID) {
			u.Status = model.UserOnline
		} else {
			u.Status = model.UserOffline
		}
		result[id] = map[string]interface{}{
			"user_id": u.ID,
			"account": u.Account,
			"name":    u.Name,
			"avatar":  u.Avatar,
			"type":    u.Type,
			"status":  u.Status,
		}
	}
	JSON(w, map[string]interface{}{"users": result})
}

type updateMeReq struct {
	Name   string `json:"name"`
	Avatar string `json:"avatar"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateMeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	if err := h.userRepo.Update(r.Context(), userID, req.Name, req.Avatar); err != nil {
		Error(w, r,http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"user_id": userID,
		"name":    req.Name,
		"avatar":  req.Avatar,
	})
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	size, _ := strconv.Atoi(r.URL.Query().Get("size"))
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	users, total, err := h.userRepo.Search(r.Context(), q, page, size)
	if err != nil {
		Error(w, r,http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	items := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		items = append(items, map[string]interface{}{
			"user_id": u.ID,
			"account": u.Account,
			"name":    u.Name,
			"avatar":  u.Avatar,
			"type":    u.Type,
			"status":  u.Status,
		})
	}
	Paginated(w, items, total, page, size)
}
