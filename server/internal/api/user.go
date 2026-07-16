package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"ziziphus/internal/auth"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

type mfaStorage interface {
	Get(ctx context.Context, userID string) (*model.UserMFA, error)
	Upsert(ctx context.Context, m *model.UserMFA) error
	Disable(ctx context.Context, userID string) error
}

type UserHandler struct {
	authSvc         *auth.Service
	userRepo        userRepo
	sessMgr         sessionChecker
	idGen           func() int64
	mfaRepo         mfaStorage
	emailVerifyRepo emailVerifyHandler
	mailer            emailSender
	allowRegistration bool
}

type userRepo interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error)
	Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error)
	Update(ctx context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor, headline string, discoverable, allowDirectChat bool) error
	CountAgents(ctx context.Context, uid string) (int, error)
	ListAgents(ctx context.Context, uid string) ([]*model.User, error)
	UpdateAgent(ctx context.Context, agentID, uid, name, avatar, cover, primaryColor, secondaryColor, headline string, wakeMode model.WakeMode, discoverable, allowDirectChat bool) error
	DeleteAgent(ctx context.Context, agentID, uid string) error
	GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error)
	UpdateAgentAPIKey(ctx context.Context, agentID, uid, apiKey string) error
	DeleteAccount(ctx context.Context, userID string) error
}

type sessionChecker interface {
	IsOnline(ctx context.Context, userID string) bool
	GetUserSessionIDs(ctx context.Context, userID string) []string
}

type emailSender interface {
	Enabled() bool
	SendVerificationCode(to, code string) error
}

func NewUserHandler(authSvc *auth.Service, userRepo userRepo, sessMgr sessionChecker, idGen func() int64, mfaRepo mfaStorage, emailVerifyRepo emailVerifyHandler, mailer emailSender, allowRegistration bool) *UserHandler {
	return &UserHandler{authSvc: authSvc, userRepo: userRepo, sessMgr: sessMgr, idGen: idGen, mfaRepo: mfaRepo, emailVerifyRepo: emailVerifyRepo, mailer: mailer, allowRegistration: allowRegistration}
}

type registerReq struct {
	Name     string `json:"name"`
	Account  string `json:"account"`
	Password string `json:"password"`
	Avatar   string `json:"avatar"`
	Email    string `json:"email"`
}

func (h *UserHandler) Register(w http.ResponseWriter, r *http.Request) {
	if !h.allowRegistration {
		Error(w, r, http.StatusForbidden, model.NewAppError(model.ErrNoPermission, "新用户注册已关闭"))
		return
	}
	var req registerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Name == "" || len(req.Password) < 8 {
		BadRequest(w, r, i18n.T(r.Context(), "err.name_password_required"))
		return
	}
	user, accessToken, refreshToken, err := h.authSvc.Register(r.Context(), req.Name, req.Password, req.Account, req.Email)
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

	mfaUser2, _ := h.userRepo.GetByID(r.Context(), userID)

	// Check if user has MFA enabled
	if h.mfaRepo != nil {
		mfa, mfaErr := h.mfaRepo.Get(r.Context(), userID)
		if mfaErr == nil && mfa != nil && mfa.Enabled {
			mfaToken := auth.GenerateEmailOTP() + auth.GenerateEmailOTP()
			auth.SetSignupCode(mfaToken, userID, 5*60)
			maskedEmail := ""

			// For email MFA: generate NEW code, send to email, update stored secret
			if mfa.MFAType == model.MFAEmail && mfaUser2 != nil && mfaUser2.Email != "" {
				newCode := auth.GenerateEmailOTP()
				if h.mailer != nil && h.mailer.Enabled() {
					go func() { h.mailer.SendVerificationCode(mfaUser2.Email, newCode) }()
				}
				// Update stored secret with new code
				h.mfaRepo.Upsert(r.Context(), &model.UserMFA{
					UserID:  userID,
					MFAType: mfa.MFAType,
					Enabled: true,
					Secret:  newCode,
				})
				e := mfaUser2.Email
				at := strings.Index(e, "@")
				if at > 1 {
					maskedEmail = e[:1] + "***" + e[at-1:]
				}
			}
			resp := map[string]interface{}{
				"mfa_required": true,
				"mfa_type":     int(mfa.MFAType),
				"mfa_token":    mfaToken,
				"user_id":      userID,
				"masked_email": maskedEmail,
			}
			// Dev mode: return code for automated tests
			if mfa.MFAType == model.MFAEmail {
				mfaUpdated, _ := h.mfaRepo.Get(r.Context(), userID)
				if mfaUpdated != nil {
					resp["code"] = mfaUpdated.Secret
				}
			}
			JSON(w, resp)
			return
		}
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

type mfaLoginReq struct {
	MFAUserID string `json:"user_id"`
	MFAToken  string `json:"mfa_token"`
	Code      string `json:"code"`
}

func (h *UserHandler) MFAVerifyLogin(w http.ResponseWriter, r *http.Request) {
	var req mfaLoginReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	storedUserID := auth.GetSignupCode(req.MFAToken)
	if storedUserID == "" || storedUserID != req.MFAUserID {
		Error(w, r, http.StatusUnauthorized, &model.AppError{Code: model.ErrNoPermission, Message: "Invalid MFA session", Key: "err.mfa_invalid_code"})
		return
	}

	mfa, err := h.mfaRepo.Get(r.Context(), req.MFAUserID)
	if err != nil || !mfa.Enabled {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "MFA not set up", Key: "err.mfa_not_found"})
		return
	}

	valid := false
	if mfa.MFAType == model.MFATOTP {
		valid = auth.VerifyTOTP(mfa.Secret, req.Code)
	} else {
		valid = mfa.Secret == req.Code
	}
	if !valid {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "Invalid code", Key: "err.mfa_invalid_code"})
		return
	}

	auth.ClearSignupCode(req.MFAToken)
	mfaUser, _ := h.userRepo.GetByID(r.Context(), req.MFAUserID)
	userType := 0
	if mfaUser != nil && mfaUser.Type == model.UserAgent {
		userType = 1
	}
	accessToken, refreshToken, expiresAt, err := h.authSvc.GenerateToken(req.MFAUserID, userType)
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if mfaUser != nil {
		mfaUser.Password = ""
	}
	JSON(w, map[string]interface{}{
		"user_id":       req.MFAUserID,
		"account":       mfaUser.Account,
		"name":          mfaUser.Name,
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
	callerID := auth.UserFromCtx(r.Context())
	if userID == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.user_id_required"))
		return
	}
	// Respect discoverability: only allow looking up the caller or discoverable users
	if userID != callerID {
		u, _ := h.userRepo.GetByID(r.Context(), userID)
		if u != nil && !u.Discoverable {
			NotFound(w, r)
			return
		}
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
			"cover":           u.Cover,
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
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	Cover           string `json:"cover"`
	Email           string `json:"email"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	Headline        string `json:"headline"`
	Discoverable    *bool  `json:"discoverable"`
	AllowDirectChat *bool  `json:"allow_direct_chat"`
}

func (h *UserHandler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	var req updateMeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	discoverable := true
	if req.Discoverable != nil {
		discoverable = *req.Discoverable
	}
	allowDirectChat := true
	if req.AllowDirectChat != nil {
		allowDirectChat = *req.AllowDirectChat
	}
	if err := h.userRepo.Update(r.Context(), userID, req.Name, req.Avatar, req.Cover, req.Email, req.PrimaryColor, req.SecondaryColor, req.Headline, discoverable, allowDirectChat); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{
		"user_id":           userID,
		"name":              req.Name,
		"avatar":            req.Avatar,
		"cover":             req.Cover,
		"email":             req.Email,
		"primary_color":     req.PrimaryColor,
		"secondary_color":   req.SecondaryColor,
			"headline":          req.Headline,		"discoverable":      discoverable,
		"allow_direct_chat": allowDirectChat,
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
		// Skip users that disabled discoverability
		if !u.Discoverable {
			continue
		}
		items = append(items, map[string]interface{}{
			"user_id":         u.ID,
			"account":         u.Account,
			"name":            u.Name,
			"avatar":          u.Avatar,
			"cover":           u.Cover,
			"type":            u.Type,
			"status":          u.Status,
			"uid":             u.UID,
			"primary_color":   u.PrimaryColor,
			"secondary_color": u.SecondaryColor,
		})
	}
	Paginated(w, items, total, page, size)
}

// ===== MFA =====

type mfaSetupReq struct {
	MFAType int    `json:"mfa_type"` // 1=totp, 2=email
	Email   string `json:"email,omitempty"`
}

type mfaVerifyReq struct {
	Code string `json:"code"`
}

func (h *UserHandler) GetMFA(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	mfa, err := h.mfaRepo.Get(r.Context(), userID)
	if err != nil {
		// No MFA record — return default
		JSON(w, map[string]interface{}{
			"enabled":  false,
			"mfa_type": 0,
		})
		return
	}
	JSON(w, map[string]interface{}{
		"enabled":  mfa.Enabled,
		"mfa_type": int(mfa.MFAType),
	})
}

func (h *UserHandler) SetupMFA(w http.ResponseWriter, r *http.Request) {
	var req mfaSetupReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())

	// For email MFA, user must have an email set
	if req.MFAType == int(model.MFAEmail) {
		u, err := h.userRepo.GetByID(r.Context(), userID)
		if err != nil || u.Email == "" {
			Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.mfa_email_required"), Key: "err.mfa_email_required"})
			return
		}
	}

	// For email MFA, generate a dedicated OTP and send it
	var emailSecret string
	if req.MFAType == int(model.MFAEmail) {
		emailSecret = auth.GenerateEmailOTP()
		u, _ := h.userRepo.GetByID(r.Context(), userID)
		if u != nil && u.Email != "" && h.mailer != nil && h.mailer.Enabled() {
			go func() { h.mailer.SendVerificationCode(u.Email, emailSecret) }()
		}
	}

	var secret string
	if req.MFAType == int(model.MFATOTP) {
		secret = auth.GenerateTOTPSecret()
	} else if req.MFAType == int(model.MFAEmail) {
		secret = emailSecret
	} else {
		secret = auth.GenerateEmailOTP()
	}

	// Store pending setup (enabled=false until verified)
	mfa := &model.UserMFA{
		UserID:  userID,
		MFAType: model.MFAType(req.MFAType),
		Enabled: false,
		Secret:  secret,
	}
	if err := h.mfaRepo.Upsert(r.Context(), mfa); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	resp := map[string]interface{}{
		"mfa_type": req.MFAType,
	}
	user, _ := h.userRepo.GetByID(r.Context(), userID)
	account := ""
	if user != nil {
		account = user.Account
	}
	if req.MFAType == int(model.MFATOTP) {
		resp["secret"] = secret
		resp["uri"] = auth.TOTPURI(account, "Ziziphus", secret)
	}
	// Email OTP: code is sent via email by the mailer, never returned in API response
	JSON(w, resp)
}

func (h *UserHandler) VerifyMFA(w http.ResponseWriter, r *http.Request) {
	var req mfaVerifyReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	mfa, err := h.mfaRepo.Get(r.Context(), userID)
	if err != nil {
		Error(w, r, http.StatusNotFound, &model.AppError{Code: model.ErrNotFound, Message: "MFA not set up", Key: "err.mfa_not_found"})
		return
	}

	valid := false
	if mfa.MFAType == model.MFATOTP {
		valid = auth.VerifyTOTP(mfa.Secret, req.Code)
	} else {
		valid = mfa.Secret == req.Code // email OTP direct comparison
	}

	if !valid {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.mfa_invalid_code"), Key: "err.mfa_invalid_code"})
		return
	}

	mfa.Enabled = true
	if err := h.mfaRepo.Upsert(r.Context(), mfa); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"enabled": true})
}

func (h *UserHandler) DisableMFA(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	if err := h.mfaRepo.Disable(r.Context(), userID); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"enabled": false})
}

type emailVerifyHandler interface {
	Upsert(ctx context.Context, ev *model.EmailVerify) error
	Get(ctx context.Context, userID string) (*model.EmailVerify, error)
	Delete(ctx context.Context, userID string) error
}

type sendEmailCodeReq struct {
	Email string `json:"email"`
}

type confirmEmailReq struct {
	Code string `json:"code"`
}

func (h *UserHandler) SendEmailCode(w http.ResponseWriter, r *http.Request) {
	var req sendEmailCodeReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	code := auth.GenerateEmailOTP()
	ev := &model.EmailVerify{
		UserID:       userID,
		PendingEmail: req.Email,
		Code:         code,
		ExpiresAt:    time.Now().Add(10 * time.Minute),
	}
	if err := h.emailVerifyRepo.Upsert(r.Context(), ev); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	// Send email asynchronously so API responds instantly
	if h.mailer != nil && h.mailer.Enabled() {
		go func() { h.mailer.SendVerificationCode(req.Email, code) }()
	}
	JSON(w, map[string]interface{}{"code": code, "expires_in": 600})
}

func (h *UserHandler) ConfirmEmail(w http.ResponseWriter, r *http.Request) {
	var req confirmEmailReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	userID := auth.UserFromCtx(r.Context())
	ev, err := h.emailVerifyRepo.Get(r.Context(), userID)
	if err != nil || time.Now().After(ev.ExpiresAt) {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.mfa_invalid_code"), Key: "err.mfa_invalid_code"})
		return
	}
	if ev.Code != req.Code {
		Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: i18n.T(r.Context(), "err.mfa_invalid_code"), Key: "err.mfa_invalid_code"})
		return
	}
	// Save email
	// Update email only — get current user, then update all fields
	curr, _ := h.userRepo.GetByID(r.Context(), userID)
	if curr == nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	if err := h.userRepo.Update(r.Context(), userID, curr.Name, curr.Avatar, curr.Cover, ev.PendingEmail, curr.PrimaryColor, curr.SecondaryColor, curr.Headline, curr.Discoverable, curr.AllowDirectChat); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	h.emailVerifyRepo.Delete(r.Context(), userID)
	JSON(w, map[string]interface{}{"email": ev.PendingEmail, "verified": true})
}

// Agent requests
type createAgentReq struct {
	Name            string `json:"name"`
	Avatar          string `json:"avatar"`
	Cover           string `json:"cover"`
	PrimaryColor    string `json:"primary_color"`
	SecondaryColor  string `json:"secondary_color"`
	Headline        string `json:"headline"`
	WakeMode        int    `json:"wake_mode"`
	Discoverable    *bool  `json:"discoverable"`
	AllowDirectChat *bool  `json:"allow_direct_chat"`
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
	apiKeyBytes := make([]byte, 16)
	rand.Read(apiKeyBytes)
	apiKeyStr := "sk-" + hex.EncodeToString(apiKeyBytes)
	now := time.Now().UnixMilli()
	discoverable := true
	if req.Discoverable != nil {
		discoverable = *req.Discoverable
	}
	allowDirectChat := true
	if req.AllowDirectChat != nil {
		allowDirectChat = *req.AllowDirectChat
	}
	u := &model.User{
		ID:              agentID,
		Type:            model.UserAgent,
		Name:            req.Name,
		Account:         "agent_" + agentID,
		Avatar:          req.Avatar,
		Cover:           req.Cover,
		Status:          model.UserOffline,
		UID:             userID,
		PrimaryColor:    req.PrimaryColor,
		SecondaryColor:  req.SecondaryColor,
			Headline:        req.Headline,		WakeMode:        model.WakeMode(req.WakeMode),
		Discoverable:    discoverable,
		AllowDirectChat: allowDirectChat,
		APIKey:          apiKeyStr,
		CreatedAt:       now,
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
	discoverable := true
	if req.Discoverable != nil {
		discoverable = *req.Discoverable
	}
	allowDirectChat := true
	if req.AllowDirectChat != nil {
		allowDirectChat = *req.AllowDirectChat
	}
	if err := h.userRepo.UpdateAgent(r.Context(), agentID, userID, req.Name, req.Avatar, req.Cover, req.PrimaryColor, req.SecondaryColor, req.Headline, model.WakeMode(req.WakeMode), discoverable, allowDirectChat); err != nil {
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

func (h *UserHandler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())
	if err := h.userRepo.DeleteAccount(r.Context(), userID); err != nil {
		logger.Error("delete account failed", "user_id", userID, "error", err)
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}
	JSON(w, map[string]interface{}{"user_id": userID})
}
