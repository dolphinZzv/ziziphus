package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/dolphinz/im-server/pkg/logger"
	"github.com/dolphinz/im-server/pkg/model"
)

type Claims struct {
	UserID string `json:"uid"`
	Type   int    `json:"typ"`
	jwt.RegisteredClaims
}

type Service struct {
	crypto   *Crypto
	jwtSecret []byte
	expireDur time.Duration
	userRepo  userRepository
}

type userRepository interface {
	Create(ctx context.Context, u *model.User) error
	GetByID(ctx context.Context, id string) (*model.User, error)
	GetByAccount(ctx context.Context, account string) (*model.User, error)
}

func NewService(crypto *Crypto, jwtSecret string, expireHours int, userRepo userRepository) *Service {
	return &Service{
		crypto:    crypto,
		jwtSecret: []byte(jwtSecret),
		expireDur: time.Duration(expireHours) * time.Hour,
		userRepo:  userRepo,
	}
}

func (s *Service) Register(ctx context.Context, name, password, account string) (*model.User, string, error) {
	if account != "" {
		existing, _ := s.userRepo.GetByAccount(ctx, account)
		if existing != nil {
			return nil, "", &model.AppError{Code: model.ErrBadMessage, Message: "账户已存在", Key: "auth.account_exists"}
		}
	}

	snowflake := model.NewSnowflake(time.Now().UnixMilli(), time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	userID := model.GenerateUserID(snowflake.NextID)

	encrypted, err := s.crypto.Encrypt(ctx, []byte(password))
	if err != nil {
		logger.Error("encrypt password failed", "error", err)
		return nil, "", fmt.Errorf("encrypt password: %w", err)
	}

	user := &model.User{
		ID:        userID,
		Account:   account,
		Type:      model.UserHuman,
		Name:      name,
		Status:    model.UserOffline,
		Password:  base64.StdEncoding.EncodeToString(encrypted),
		CreatedAt: time.Now().UnixMilli(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", fmt.Errorf("create user: %w", err)
	}

	token, err := s.generateToken(userID, int(model.UserHuman))
	if err != nil {
		return nil, "", err
	}
	user.Password = ""
	return user, token, nil
}

func (s *Service) Login(ctx context.Context, account, password string) (string, int64, string, error) {
	user, err := s.userRepo.GetByAccount(ctx, account)
	if err != nil {
		logger.Info("Login user not found", "account", account, "error", err)
		return "", 0, "", &model.AppError{Code: model.ErrNoPermission, Message: "用户不存在", Key: "auth.user_not_found"}
	}

	var ciphertext []byte
	ciphertext, err = base64.StdEncoding.DecodeString(user.Password)
	if err != nil {
		return "", 0, "", fmt.Errorf("decode password: %w", err)
	}
	decrypted, err := s.crypto.Decrypt(ctx, ciphertext)
	if err != nil {
		return "", 0, "", fmt.Errorf("decrypt password: %w", err)
	}
	if string(decrypted) != password {
		return "", 0, "", &model.AppError{Code: model.ErrNoPermission, Message: "密码错误", Key: "auth.wrong_password"}
	}

	token, err := s.generateToken(user.ID, int(user.Type))
	if err != nil {
		return "", 0, "", err
	}
	return token, time.Now().Add(s.expireDur).Unix(), user.ID, nil
}

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
	return claims, nil
}

func (s *Service) generateToken(userID string, userType int) (string, error) {
	claims := &Claims{
		UserID: userID,
		Type:   userType,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(s.expireDur)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "im-server",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
}
