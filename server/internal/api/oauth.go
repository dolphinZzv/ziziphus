package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
	"ziziphus/internal/auth"
	"ziziphus/pkg/i18n"
	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

type OAuthHandler struct {
	oauthSvc    *auth.OAuthService
	authSvc     tokenGenerator
	authMW      func(http.Handler) http.Handler
	frontendURL string
}

type tokenGenerator interface {
	GenerateToken(userID string, userType int) (string, string, int64, error)
	GenerateFileToken(ctx context.Context, userID string) (string, error)
}

func NewOAuthHandler(oauthSvc *auth.OAuthService, authSvc tokenGenerator, authMW func(http.Handler) http.Handler, frontendURL string) *OAuthHandler {
	return &OAuthHandler{oauthSvc: oauthSvc, authSvc: authSvc, authMW: authMW, frontendURL: frontendURL}
}

func (h *OAuthHandler) frontendBase() string {
	if h.frontendURL != "" {
		return h.frontendURL
	}
	return ""
}

// GET /api/v1/auth/{provider}/login
func (h *OAuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")

	state, err := generateState()
	if err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	bindUserID := r.URL.Query().Get("bind")

	authURL, err := h.oauthSvc.GetAuthorizationURL(provider, state, bindUserID)
	if err != nil {
		msg := err.Error()
		switch msg {
		case "oauth provider disabled":
			Error(w, r, http.StatusForbidden, &model.AppError{Code: model.ErrNoPermission, Message: "OAuth provider disabled", Key: "err.oauth_disabled"})
		case "unsupported oauth provider: " + provider:
			Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "Unsupported provider", Key: "err.oauth_invalid_provider"})
		default:
			Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: msg, Key: "err.oauth_invalid_provider"})
		}
		return
	}

	JSON(w, map[string]any{
		"url": authURL,
	})
}

// GET /api/v1/auth/{provider}/callback
func (h *OAuthHandler) Callback(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	code := r.URL.Query().Get("code")
	stateParam := r.URL.Query().Get("state")
	redirectErr := r.URL.Query().Get("error")

	if redirectErr != "" {
		h.redirectError(w, r, fmt.Sprintf("provider_error:%s", redirectErr))
		return
	}

	if code == "" || stateParam == "" {
		h.redirectError(w, r, "missing_params")
		return
	}

	state := h.oauthSvc.StateStore().GetAndClear(stateParam)
	if state == nil {
		h.redirectError(w, r, "invalid_state")
		return
	}

	if state.Provider != provider {
		h.redirectError(w, r, "provider_mismatch")
		return
	}

	info, err := h.oauthSvc.ExchangeCode(r.Context(), provider, code)
	if err != nil {
		logger.Error("oauth code exchange failed", "provider", provider, "error", err)
		h.redirectError(w, r, "exchange_failed")
		return
	}

	if state.Mode == "bind" {
		if state.UserID == "" {
			h.redirectError(w, r, "missing_user")
			return
		}
		if err := h.oauthSvc.BindUser(r.Context(), state.UserID, provider, info.ID); err != nil {
			logger.Error("oauth bind failed", "user_id", state.UserID, "error", err)
			h.redirectError(w, r, "bind_failed")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, `<html><script>
window.opener.postMessage({type:"oauth_bound", provider:"%s"}, "*");
window.close();
</script></html>`, provider)
		return
	}

	user, isNew, err := h.oauthSvc.FindOrCreateUser(r.Context(), info)
	if err != nil {
		logger.Error("oauth find or create user failed", "provider", provider, "error", err)
		h.redirectError(w, r, fmt.Sprintf("user_creation_failed: %s", err.Error()))
		return
	}

	accessToken, refreshToken, _, err := h.authSvc.GenerateToken(user.ID, int(user.Type))
	if err != nil {
		logger.Error("oauth generate token failed", "user_id", user.ID, "error", err)
		h.redirectError(w, r, "token_generation_failed")
		return
	}

	fileToken, _ := h.authSvc.GenerateFileToken(r.Context(), user.ID)

	base := h.frontendBase()
	callbackURL := fmt.Sprintf("%s/oauth/callback?token=%s&refresh_token=%s&file_token=%s&user_id=%s&name=%s&is_new=%t",
		base, url.QueryEscape(accessToken), url.QueryEscape(refreshToken), url.QueryEscape(fileToken), url.QueryEscape(user.ID), url.QueryEscape(user.Name), isNew)

	http.Redirect(w, r, callbackURL, http.StatusFound)
}

func (h *OAuthHandler) redirectError(w http.ResponseWriter, r *http.Request, errorMsg string) {
	base := h.frontendBase()
	callbackURL := fmt.Sprintf("%s/oauth/callback?error=%s", base, url.QueryEscape(errorMsg))
	http.Redirect(w, r, callbackURL, http.StatusFound)
}

// POST /api/v1/users/me/oauth/bind
func (h *OAuthHandler) Bind(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())

	var req struct {
		Provider          string `json:"provider"`
		AuthorizationCode string `json:"authorization_code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Provider == "" || req.AuthorizationCode == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	info, err := h.oauthSvc.ExchangeCode(r.Context(), req.Provider, req.AuthorizationCode)
	if err != nil {
		Error(w, r, http.StatusBadGateway, &model.AppError{Code: model.ErrBadMessage, Message: "OAuth exchange failed: " + err.Error(), Key: "err.oauth_exchange_failed"})
		return
	}

	if err := h.oauthSvc.BindUser(r.Context(), userID, req.Provider, info.ID); err != nil {
		if err.Error() == "github account already bound to another user" || err.Error() == "google account already bound to another user" {
			Error(w, r, http.StatusConflict, &model.AppError{Code: model.ErrBadMessage, Message: "Account already bound", Key: "err.oauth_already_bound"})
			return
		}
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]any{
		"status":         "ok",
		"bound_provider": req.Provider,
		"bound_id":       info.ID,
	})
}

// POST /api/v1/users/me/oauth/unbind
func (h *OAuthHandler) Unbind(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserFromCtx(r.Context())

	var req struct {
		Provider string `json:"provider"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}
	if req.Provider == "" {
		BadRequest(w, r, i18n.T(r.Context(), "err.invalid_params"))
		return
	}

	// Protection: check if user has password set
	user, err := h.oauthSvc.GetUser(r.Context(), userID)
	if err != nil {
		NotFound(w, r)
		return
	}
	if !auth.HasPasswordFromOAuth(user) {
		hasOtherBinding := false
		switch req.Provider {
		case "github":
			hasOtherBinding = user.GoogleID != ""
		case "google":
			hasOtherBinding = user.GithubID != ""
		}
		if !hasOtherBinding {
			Error(w, r, http.StatusBadRequest, &model.AppError{Code: model.ErrBadMessage, Message: "Cannot unbind last login method. Set a password first.", Key: "err.oauth_last_login_method"})
			return
		}
	}

	if err := h.oauthSvc.UnbindUser(r.Context(), userID, req.Provider); err != nil {
		Error(w, r, http.StatusInternalServerError, model.ErrInternalServer)
		return
	}

	JSON(w, map[string]any{"status": "ok"})
}

func generateState() (string, error) {
	return auth.GenerateEmailOTP() + auth.GenerateEmailOTP(), nil
}
