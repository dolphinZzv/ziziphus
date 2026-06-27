package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
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

type UserHandler struct {
	authSvc  *auth.Service
	userRepo userRepo
	sessMgr  sessionChecker
	idGen    func() int64
}

type userRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
	Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error)
	Update(ctx context.Context, id, name, avatar, primaryColor, secondaryColor string) error
	CountAgents(ctx context.Context, uid string) (int, error)
	ListAgents(ctx context.Context, uid string) ([]*model.User, error)
	UpdateAgent(ctx context.Context, agentID, uid, name, avatar, primaryColor, secondaryColor string, wakeMode model.WakeMode) error
	DeleteAgent(ctx context.Context, agentID, uid string) error
	GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error)
	UpdateAgentAPIKey(ctx context.Context, agentID, uid, apiKey string) error
}

type sessionChecker interface {
	IsOnline(ctx context.Context, userID string) bool
	GetUserSessionIDs(ctx context.Context, userID string) []string
}

func NewUserHandler(authSvc *auth.Service, userRepo userRepo, sessMgr sessionChecker, idGen func() int64) *UserHandler {
	return &UserHandler{authSvc: authSvc, userRepo: userRepo, sessMgr: sessMgr, idGen: idGen}
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
	user, accessToken, refreshToken, err := h.authSvc.Register(r.Context(), req.Name, req.Password, req.Account)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusBadRequest, appErr)
			return
		}
		logger.Error("register failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"user_id":       user.ID,
		"account":       user.Account,
		"name":          user.Name,
		"token":         accessToken,
		"refresh_token": refreshToken,
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
	accessToken, refreshToken, expiresAt, userID, err := h.authSvc.Login(r.Context(), req.Account, req.Password)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusUnauthorized, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	user, _ := h.userRepo.GetByID(r.Context(), userID)
	if user != nil {
		user.Password = ""
	}
	JSON(w, map[string]interface{}{
		"user_id":       userID,
		"account":       req.Account,
		"name":          user.Name,
		"token":         accessToken,
		"refresh_token": refreshToken,
		"expires_at":    expiresAt,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (h *UserHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.RefreshToken == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	accessToken, expiresAt, err := h.authSvc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		if appErr, ok := err.(*model.AppError); ok {
			Error(w, r, http.StatusUnauthorized, appErr)
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"token":      accessToken,
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
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
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
			"user_id":         u.ID,
			"account":         u.Account,
			"name":            u.Name,
			"avatar":          u.Avatar,
			"type":            u.Type,
			"status":          u.Status,
			"uid":             u.UID,
			"primary_color":   u.PrimaryColor,
			"secondary_color": u.SecondaryColor,
		}
	}
	JSON(w, map[string]interface{}{"users": result})
}

type updateMeReq struct {
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateMeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	if err := h.userRepo.Update(r.Context(), userID, req.Name, req.Avatar, req.PrimaryColor, req.SecondaryColor); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"user_id":         userID,
		"name":            req.Name,
		"avatar":          req.Avatar,
		"primary_color":   req.PrimaryColor,
		"secondary_color": req.SecondaryColor,
	})
}

func (h *UserHandler) Search(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	if len(q) < 2 {
		Paginated(w, []map[string]interface{}{}, 0, 1, 20)
		return
	}
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
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	items := make([]map[string]interface{}, 0, len(users))
	for _, u := range users {
		items = append(items, map[string]interface{}{
			"user_id":         u.ID,
			"account":         u.Account,
			"name":            u.Name,
			"avatar":          u.Avatar,
			"type":            u.Type,
			"status":          u.Status,
			"uid":             u.UID,
			"primary_color":   u.PrimaryColor,
			"secondary_color": u.SecondaryColor,
		})
	}
	Paginated(w, items, total, page, size)
}

// Agent requests
type createAgentReq struct {
	Name           string `json:"name"`
	Avatar         string `json:"avatar"`
	PrimaryColor   string `json:"primary_color"`
	SecondaryColor string `json:"secondary_color"`
	WakeMode       int    `json:"wake_mode"`
}

func (h *UserHandler) ListMyAgents(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	agents, err := h.userRepo.ListAgents(r.Context(), userID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if agents == nil {
		agents = []*model.User{}
	}
	JSON(w, agents)
}

func (h *UserHandler) CreateAgent(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	var req createAgentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Name == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.name_password_required"))
		return
	}

	// Check limit
	count, err := h.userRepo.CountAgents(r.Context(), userID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if count >= 10 {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "agent limit reached", Key: "err.agent_limit"})
		return
	}

	agentID := model.GenerateUserID(h.idGen)
	now := time.Now().UnixMilli()
	u := &model.User{
		ID:             agentID,
		Type:           model.UserAgent,
		Name:           req.Name,
		Account:        "agent_" + agentID,
		Avatar:         req.Avatar,
		Status:         model.UserOffline,
		UID:            userID,
		PrimaryColor:   req.PrimaryColor,
		SecondaryColor: req.SecondaryColor,
		WakeMode:       model.WakeMode(req.WakeMode),
		CreatedAt:      now,
	}
	if err := h.userRepo.Create(r.Context(), u); err != nil {
		logger.Error("create agent failed", "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, u)
}

func (h *UserHandler) UpdateAgent(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	agentID := chi.URLParam(r, "agent_id")
	if agentID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	var req createAgentReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if err := h.userRepo.UpdateAgent(r.Context(), agentID, userID, req.Name, req.Avatar, req.PrimaryColor, req.SecondaryColor, model.WakeMode(req.WakeMode)); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}

// RegenerateAgentKey regenerates the api_key for an agent.
func (h *UserHandler) RegenerateAgentKey(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	agentID := chi.URLParam(r, "agent_id")
	if agentID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	// Verify agent belongs to user
	agents, err := h.userRepo.ListAgents(r.Context(), userID)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	found := false
	for _, a := range agents {
		if a.ID == agentID {
			found = true
			break
		}
	}
	if !found {
		NotFound(w, r)
		return
	}

	apiKeyBytes := make([]byte, 16)
	rand.Read(apiKeyBytes)
	apiKey := "sk-" + hex.EncodeToString(apiKeyBytes)
	if err := h.userRepo.UpdateAgentAPIKey(r.Context(), agentID, userID, apiKey); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"api_key": apiKey})
}

func (h *UserHandler) DeleteAgent(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	agentID := chi.URLParam(r, "agent_id")
	if agentID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if err := h.userRepo.DeleteAgent(r.Context(), agentID, userID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]string{"status": "ok"})
}
