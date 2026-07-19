package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"ziziphus/config"
	"ziziphus/pkg/model"
)

const (
	OAuthScopeGitHub = "read:user user:email"
	OAuthScopeGoogle = "openid profile email"
)

type OAuthUserInfo struct {
	Provider string
	ID       string
	Name     string
	Email    string
	Avatar   string
}

type OAuthState struct {
	Provider  string    `json:"provider"`
	Mode      string    `json:"mode"` // "login" | "bind"
	UserID    string    `json:"user_id,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
}

type OAuthStateStore struct {
	mu  sync.RWMutex
	mem map[string]*OAuthState
}

func NewOAuthStateStore() *OAuthStateStore {
	return &OAuthStateStore{mem: make(map[string]*OAuthState)}
}

func (s *OAuthStateStore) Set(state string, data *OAuthState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data.ExpiresAt = time.Now().Add(10 * time.Minute)
	s.mem[state] = data
}

func (s *OAuthStateStore) GetAndClear(state string) *OAuthState {
	s.mu.Lock()
	defer s.mu.Unlock()
	data := s.mem[state]
	delete(s.mem, state)
	if data != nil && time.Now().After(data.ExpiresAt) {
		return nil
	}
	return data
}

type TokenGenerator interface {
	GenerateToken(userID string, userType int) (string, string, int64, error)
	GenerateFileToken(ctx context.Context, userID string) (string, error)
}

type OAuthUserRepo interface {
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByGithubID(ctx context.Context, githubID string) (*model.User, error)
	GetByGoogleID(ctx context.Context, googleID string) (*model.User, error)
	GetByAccount(ctx context.Context, account string) (*model.User, error)
	GetByEmail(ctx context.Context, email string) (*model.User, error)
	Create(ctx context.Context, u *model.User) error
	UpdateOAuthID(ctx context.Context, userID, provider, oauthID string) error
	ClearOAuthID(ctx context.Context, userID, provider string) error
}

type OAuthService struct {
	cfg        config.OAuthConfig
	httpClient *http.Client
	idGen      func() int64
	svc        TokenGenerator
	userRepo   OAuthUserRepo
	stateStore *OAuthStateStore
}

func NewOAuthService(cfg config.OAuthConfig, idGen func() int64, svc TokenGenerator, userRepo OAuthUserRepo) *OAuthService {
	return &OAuthService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		idGen:      idGen,
		svc:        svc,
		userRepo:   userRepo,
		stateStore: NewOAuthStateStore(),
	}
}

func (s *OAuthService) StateStore() *OAuthStateStore {
	return s.stateStore
}

func secureRandHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GitHub OAuth App 的 scope 必须勾选 read:user 和 user:email

func (s *OAuthService) GetAuthorizationURL(provider, state, bindUserID string) (string, error) {
	var authURL, clientID, scope string
	switch provider {
	case "github":
		if !s.cfg.GitHub.Enabled {
			return "", fmt.Errorf("oauth provider disabled")
		}
		clientID = s.cfg.GitHub.ClientID
		scope = OAuthScopeGitHub
		authURL = "https://github.com/login/oauth/authorize"
	case "google":
		if !s.cfg.Google.Enabled {
			return "", fmt.Errorf("oauth provider disabled")
		}
		clientID = s.cfg.Google.ClientID
		scope = OAuthScopeGoogle
		authURL = "https://accounts.google.com/o/oauth2/v2/auth"
	default:
		return "", fmt.Errorf("unsupported oauth provider: %s", provider)
	}

	redirectURI := s.redirectURL(provider)
	mode := "login"
	if bindUserID != "" {
		mode = "bind"
	}

	s.stateStore.Set(state, &OAuthState{
		Provider: provider,
		Mode:     mode,
		UserID:   bindUserID,
	})

	q := url.Values{}
	q.Set("client_id", clientID)
	q.Set("redirect_uri", redirectURI)
	q.Set("state", state)
	q.Set("response_type", "code")
	q.Set("scope", scope)
	if provider == "google" {
		q.Set("access_type", "online")
	}

	return authURL + "?" + q.Encode(), nil
}

func (s *OAuthService) redirectURL(provider string) string {
	switch provider {
	case "github":
		return s.cfg.GitHub.RedirectURL
	case "google":
		return s.cfg.Google.RedirectURL
	}
	return ""
}

func (s *OAuthService) ExchangeCode(ctx context.Context, provider, code string) (*OAuthUserInfo, error) {
	switch provider {
	case "github":
		return s.exchangeGitHubCode(ctx, code)
	case "google":
		return s.exchangeGoogleCode(ctx, code)
	}
	return nil, fmt.Errorf("unsupported oauth provider: %s", provider)
}

func (s *OAuthService) exchangeGitHubCode(ctx context.Context, code string) (*OAuthUserInfo, error) {
	tokenURL := "https://github.com/login/oauth/access_token"
	redirectURI := s.redirectURL("github")

	data := url.Values{}
	data.Set("client_id", s.cfg.GitHub.ClientID)
	data.Set("client_secret", s.cfg.GitHub.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)

	req, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("github token exchange: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Error       string `json:"error"`
		ErrorDesc   string `json:"error_description"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("github token response decode: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("github oauth error: %s - %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	// Fetch user info
	userReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
	userReq.Header.Set("Accept", "application/json")

	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("github user fetch: %w", err)
	}
	defer userResp.Body.Close()

	var ghUser struct {
		ID        int    `json:"id"`
		Login     string `json:"login"`
		Name      string `json:"name"`
		Email     string `json:"email"`
		AvatarURL string `json:"avatar_url"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&ghUser); err != nil {
		return nil, fmt.Errorf("github user decode: %w", err)
	}

	// If email is empty, try fetching emails endpoint
	email := ghUser.Email
	if email == "" {
		emailReq, _ := http.NewRequestWithContext(ctx, "GET", "https://api.github.com/user/emails", nil)
		emailReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		emailReq.Header.Set("Accept", "application/json")

		emailResp, err := s.httpClient.Do(emailReq)
		if err == nil {
			defer emailResp.Body.Close()
			var emails []struct {
				Email    string `json:"email"`
				Primary  bool   `json:"primary"`
				Verified bool   `json:"verified"`
			}
			if json.NewDecoder(emailResp.Body).Decode(&emails) == nil {
				for _, e := range emails {
					if e.Primary && e.Verified {
						email = e.Email
						break
					}
				}
			}
		}
	}

	name := ghUser.Name
	if name == "" {
		name = ghUser.Login
	}

	return &OAuthUserInfo{
		Provider: "github",
		ID:       fmt.Sprintf("%d", ghUser.ID),
		Name:     name,
		Email:    email,
		Avatar:   ghUser.AvatarURL,
	}, nil
}

func (s *OAuthService) exchangeGoogleCode(ctx context.Context, code string) (*OAuthUserInfo, error) {
	tokenURL := "https://oauth2.googleapis.com/token"
	redirectURI := s.redirectURL("google")

	data := url.Values{}
	data.Set("client_id", s.cfg.Google.ClientID)
	data.Set("client_secret", s.cfg.Google.ClientSecret)
	data.Set("code", code)
	data.Set("redirect_uri", redirectURI)
	data.Set("grant_type", "authorization_code")

	req, _ := http.NewRequestWithContext(ctx, "POST", tokenURL, strings.NewReader(data.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google token exchange: %w", err)
	}
	defer resp.Body.Close()

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		IDToken     string `json:"id_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, fmt.Errorf("google token response decode: %w", err)
	}
	if tokenResp.Error != "" {
		return nil, fmt.Errorf("google oauth error: %s", tokenResp.Error)
	}

	// Fetch user info
	userReq, _ := http.NewRequestWithContext(ctx, "GET", "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)

	userResp, err := s.httpClient.Do(userReq)
	if err != nil {
		return nil, fmt.Errorf("google user fetch: %w", err)
	}
	defer userResp.Body.Close()

	var gUser struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Email   string `json:"email"`
		Picture string `json:"picture"`
	}
	if err := json.NewDecoder(userResp.Body).Decode(&gUser); err != nil {
		return nil, fmt.Errorf("google user decode: %w", err)
	}

	return &OAuthUserInfo{
		Provider: "google",
		ID:       gUser.ID,
		Name:     gUser.Name,
		Email:    gUser.Email,
		Avatar:   gUser.Picture,
	}, nil
}

func (s *OAuthService) FindOrCreateUser(ctx context.Context, info *OAuthUserInfo) (*model.User, bool, error) {
	var existing *model.User
	var err error
	switch info.Provider {
	case "github":
		existing, err = s.userRepo.GetByGithubID(ctx, info.ID)
	case "google":
		existing, err = s.userRepo.GetByGoogleID(ctx, info.ID)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, false, err
	}
	if existing != nil {
		existing.Password = ""
		return existing, false, nil
	}

	// Email conflict handling
	email := info.Email
	if email != "" {
		existingEmail, _ := s.userRepo.GetByEmail(ctx, email)
		if existingEmail != nil {
			email = ""
		}
	}

	randPass := make([]byte, 32)
	if _, err := rand.Read(randPass); err != nil {
		return nil, false, fmt.Errorf("rand read: %w", err)
	}
	hashed, err := HashPassword(string(randPass))
	if err != nil {
		return nil, false, fmt.Errorf("hash password: %w", err)
	}

	userID := model.GenerateUserID(s.idGen)
	account := fmt.Sprintf("%s_%s", info.Provider, info.ID)

	// Account conflict fallback
	if existingAccount, _ := s.userRepo.GetByAccount(ctx, account); existingAccount != nil {
		suffix, _ := secureRandHex(4)
		account = fmt.Sprintf("%s_%s", info.Provider, suffix)
	}

	user := &model.User{
		ID:              userID,
		Type:            model.UserHuman,
		Name:            info.Name,
		Email:           email,
		Avatar:          info.Avatar,
		Password:        string(hashed),
		Account:         account,
		Status:          model.UserOffline,
		Discoverable:    true,
		AllowDirectChat: true,
		CreatedAt:       time.Now().UnixMilli(),
	}
	switch info.Provider {
	case "github":
		user.GithubID = info.ID
	case "google":
		user.GoogleID = info.ID
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, false, fmt.Errorf("create user: %w", err)
	}

	user.Password = ""
	return user, true, nil
}

func (s *OAuthService) BindUser(ctx context.Context, userID, provider, oauthID string) error {
	// Check if already bound by another user
	switch provider {
	case "github":
		existing, _ := s.userRepo.GetByGithubID(ctx, oauthID)
		if existing != nil && existing.ID != userID {
			return fmt.Errorf("github account already bound to another user")
		}
	case "google":
		existing, _ := s.userRepo.GetByGoogleID(ctx, oauthID)
		if existing != nil && existing.ID != userID {
			return fmt.Errorf("google account already bound to another user")
		}
	}
	return s.userRepo.UpdateOAuthID(ctx, userID, provider, oauthID)
}

func (s *OAuthService) UnbindUser(ctx context.Context, userID, provider string) error {
	return s.userRepo.ClearOAuthID(ctx, userID, provider)
}

func (s *OAuthService) GetUser(ctx context.Context, userID string) (*model.User, error) {
	return s.userRepo.GetByID(ctx, userID)
}

// HasPasswordFromOAuth returns true if the user has a non-random (user-set) password.
// OAuth-created users get a random bcrypt hash; this checks if the password
// was actually set by the user.
func HasPasswordFromOAuth(user *model.User) bool {
	return hasBcryptPrefix(user.Password)
}

func hasBcryptPrefix(pw string) bool {
	if len(pw) < 4 {
		return false
	}
	return pw[:4] == "$2a$" || pw[:4] == "$2b$"
}
