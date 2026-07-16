package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/golang-jwt/jwt/v5"
	"ziziphus/pkg/model"
)

// Claims represents the JWT claims for access tokens.
type Claims struct {
	UserID string `json:"uid"`
	Type   int    `json:"typ"`
	jwt.RegisteredClaims
}

// refreshTokenClaims contains the claims for refresh tokens (opaque, stored in Redis).
type refreshTokenData struct {
	UserID    string `json:"uid"`
	TokenID   string `json:"tid"`
	ExpiresAt int64  `json:"exp"`
}

// Service handles authentication, including password hashing, JWT access tokens,
// and refresh tokens stored in Redis.
type Service struct {
	jwtSecret     []byte
	accessExpire  time.Duration
	refreshExpire time.Duration
	userRepo      userRepository
	rdb           redis.UniversalClient
	idGen         func() int64
}

type userRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByAccount(ctx context.Context, account string) (*model.User, error)
}

const (
	refreshTokenKeyPrefix = "refresh_token:"
	blacklistKeyPrefix    = "token_blacklist:"
)

// NewService creates a new auth Service.
// If rdb is nil, refresh token and blacklist features are disabled.
func NewService(jwtSecret string, accessExpireHours, refreshExpireHours int, userRepo userRepository, rdb redis.UniversalClient, idGen func() int64) *Service {
	return &Service{
		jwtSecret:     []byte(jwtSecret),
		accessExpire:  time.Duration(accessExpireHours) * time.Hour,
		refreshExpire: time.Duration(refreshExpireHours) * time.Hour,
		userRepo:      userRepo,
		rdb:           rdb,
		idGen:         idGen,
	}
}

// Register creates a new user with a bcrypt-hashed password and returns tokens.
func (s *Service) Register(ctx context.Context, name, password, account, email, language string) (*model.User, string, string, error) {
	if len(password) < 8 {
		return nil, "", "", &model.AppError{Code: model.ErrBadMessage, Message: "password must be at least 8 characters", Key: "auth.password_too_short"}
	}
	if account != "" {
		existing, _ := s.userRepo.GetByAccount(ctx, account)
		if existing != nil {
			return nil, "", "", &model.AppError{Code: model.ErrBadMessage, Message: "account already exists", Key: "auth.account_exists"}
		}
	}

	userID := model.GenerateUserID(s.idGen)

	hashed, err := HashPassword(password)
	if err != nil {
		return nil, "", "", fmt.Errorf("hash password: %w", err)
	}

	user := &model.User{
		ID:              userID,
		Account:         account,
		Type:            model.UserHuman,
		Name:            name,
		Email:           email,
		Status:          model.UserOffline,
		Password:        hashed,
		Discoverable:    true,
		AllowDirectChat: true,
		CreatedAt:       time.Now().UnixMilli(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", "", fmt.Errorf("create user: %w", err)
	}

	accessToken, err := s.generateAccessToken(userID, int(model.UserHuman))
	if err != nil {
		return nil, "", "", err
	}
	refreshToken, err := s.generateRefreshToken(ctx, userID)
	if err != nil {
		return nil, "", "", err
	}

	user.Password = ""
	return user, accessToken, refreshToken, nil
}

// Login verifies credentials and returns tokens.
func (s *Service) Login(ctx context.Context, account, password string) (string, string, int64, string, error) {
	user, err := s.userRepo.GetByAccount(ctx, account)
	if err != nil {
		return "", "", 0, "", &model.AppError{Code: model.ErrNoPermission, Message: "invalid account or password", Key: "auth.bad_credentials"}
	}

	if !CheckPassword(password, user.Password) {
		return "", "", 0, "", &model.AppError{Code: model.ErrNoPermission, Message: "invalid account or password", Key: "auth.bad_credentials"}
	}

	// Check if user is banned
	if user.Banned {
		return "", "", 0, "", model.ErrUserBanned
	}

	accessToken, err := s.generateAccessToken(user.ID, int(user.Type))
	if err != nil {
		return "", "", 0, "", err
	}
	refreshToken, err := s.generateRefreshToken(ctx, user.ID)
	if err != nil {
		return "", "", 0, "", err
	}

	return accessToken, refreshToken, time.Now().Add(s.accessExpire).Unix(), user.ID, nil
}

// GenerateToken creates tokens for a given user ID (used by MFA flow).
func (s *Service) GenerateToken(userID string, userType int) (string, string, int64, error) {
	accessToken, err := s.generateAccessToken(userID, userType)
	if err != nil {
		return "", "", 0, err
	}
	refreshToken, err := s.generateRefreshToken(context.Background(), userID)
	if err != nil {
		return "", "", 0, err
	}
	return accessToken, refreshToken, time.Now().Add(s.accessExpire).Unix(), nil
}

// RefreshToken validates a refresh token and returns a new access token.
func (s *Service) RefreshToken(ctx context.Context, refreshTokenStr string) (string, int64, error) {
	if s.rdb == nil {
		return "", 0, fmt.Errorf("refresh tokens not available")
	}

	data, err := s.rdb.Get(ctx, refreshTokenKeyPrefix+refreshTokenStr).Bytes()
	if err == redis.Nil {
		return "", 0, &model.AppError{Code: model.ErrNoPermission, Message: "refresh token invalid or expired", Key: "auth.invalid_refresh_token"}
	}
	if err != nil {
		return "", 0, fmt.Errorf("get refresh token: %w", err)
	}

	var rt refreshTokenData
	if err := json.Unmarshal(data, &rt); err != nil {
		return "", 0, fmt.Errorf("unmarshal refresh token: %w", err)
	}

	// Delete the used refresh token (rotation)
	s.rdb.Del(ctx, refreshTokenKeyPrefix+refreshTokenStr)

	if rt.ExpiresAt < time.Now().Unix() {
		return "", 0, &model.AppError{Code: model.ErrNoPermission, Message: "refresh token expired", Key: "auth.invalid_refresh_token"}
	}

	accessToken, err := s.generateAccessToken(rt.UserID, 0)
	if err != nil {
		return "", 0, err
	}

	return accessToken, time.Now().Add(s.accessExpire).Unix(), nil
}

// ParseToken parses and validates an access token, checking the blacklist.
func (s *Service) ParseToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	// Check blacklist if Redis is available
	if s.rdb != nil {
		blacklisted, err := s.rdb.Exists(ctxTODO, blacklistKeyPrefix+claims.ID).Result()
		if err == nil && blacklisted > 0 {
			return nil, fmt.Errorf("token has been revoked")
		}
	}

	return claims, nil
}

func (s *Service) generateAccessToken(userID string, userType int) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID: userID,
		Type:   userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessExpire)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "ziziphus",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}

// generateRefreshToken creates an opaque refresh token stored in Redis.
func (s *Service) generateRefreshToken(ctx context.Context, userID string) (string, error) {
	if s.rdb == nil {
		// Fallback: return empty string if Redis is not configured
		return "", nil
	}

	tokenID := make([]byte, 32)
	if _, err := rand.Read(tokenID); err != nil {
		return "", fmt.Errorf("generate refresh token: %w", err)
	}
	tokenStr := hex.EncodeToString(tokenID)

	rt := refreshTokenData{
		UserID:    userID,
		TokenID:   tokenStr,
		ExpiresAt: time.Now().Add(s.refreshExpire).Unix(),
	}
	data, err := json.Marshal(rt)
	if err != nil {
		return "", fmt.Errorf("marshal refresh token: %w", err)
	}

	if err := s.rdb.Set(ctx, refreshTokenKeyPrefix+tokenStr, data, s.refreshExpire).Err(); err != nil {
		return "", fmt.Errorf("store refresh token: %w", err)
	}

	return tokenStr, nil
}

// ctxTODO is a background context for blacklist checks where the request
// context isn't available. Only used in ParseToken for blacklist lookups.
var ctxTODO = context.Background()
