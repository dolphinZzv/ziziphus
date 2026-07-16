package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
	"ziziphus/pkg/model"
)

// setupServiceWithRedis creates a Service backed by miniredis for refresh token tests.
func setupServiceWithRedis(t *testing.T) (*Service, *mockUserRepo, *miniredis.Miniredis) {
	t.Helper()
	mr := miniredis.NewMiniRedis()
	if err := mr.Start(); err != nil {
		t.Fatalf("miniredis.Start: %v", err)
	}
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { rdb.Close() })

	repo := newMockUserRepo()
	svc := NewService("test-jwt-secret", 24, 168, repo, rdb, func() int64 { return time.Now().UnixNano() })
	return svc, repo, mr
}

func registerForRefresh(svc *Service, name, account, password string) (string, error) {
	ctx := context.Background()
	_, _, refreshToken, err := svc.Register(ctx, name, password, account, "")
	return refreshToken, err
}

// ---------------------------------------------------------------------------
// RefreshToken
// ---------------------------------------------------------------------------

func TestRefreshToken_Success(t *testing.T) {
	svc, _, mr := setupServiceWithRedis(t)
	ctx := context.Background()

	_, _, refreshToken, err := svc.Register(ctx, "alice", "p@ssword!", "alice_rf", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	newAccessToken, expireAt, err := svc.RefreshToken(ctx, refreshToken)
	if err != nil {
		t.Fatalf("RefreshToken failed: %v", err)
	}
	now := time.Now().Unix()
	if expireAt <= now {
		t.Errorf("expireAt (%d) should be in the future", expireAt)
	}

	// Verify old refresh token was deleted (rotation)
	if mr.Exists("refresh_token:" + refreshToken) {
		t.Error("old refresh token should have been deleted after rotation")
	}

	// New token should parse
	claims, err := svc.ParseToken(newAccessToken)
	if err != nil {
		t.Fatalf("ParseToken(new token): %v", err)
	}
	if claims.UserID == "" {
		t.Error("expected non-empty UserID in new token claims")
	}
}

func TestRefreshToken_NilRdb(t *testing.T) {
	svc, _, _ := setupService(t)
	ctx := context.Background()

	_, _, err := svc.RefreshToken(ctx, "some-token")
	if err == nil {
		t.Fatal("expected error for nil rdb, got nil")
	}
}

func TestRefreshToken_InvalidToken(t *testing.T) {
	svc, _, _ := setupServiceWithRedis(t)
	ctx := context.Background()

	_, _, err := svc.RefreshToken(ctx, "nonexistent-token")
	if err == nil {
		t.Fatal("expected error for invalid refresh token, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
	if appErr.Code != model.ErrNoPermission {
		t.Errorf("AppError.Code = %d, want %d", appErr.Code, model.ErrNoPermission)
	}
}

func TestRefreshToken_Expired(t *testing.T) {
	svc, _, mr := setupServiceWithRedis(t)
	ctx := context.Background()

	_, _, refreshToken, err := svc.Register(ctx, "bob", "password", "bob_exp", "")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}

	// Manually expire the token in miniredis
	mr.FastForward(200 * 24 * time.Hour) // past the 168 h refresh expiry

	_, _, err = svc.RefreshToken(ctx, refreshToken)
	if err == nil {
		t.Fatal("expected error for expired refresh token, got nil")
	}
	var appErr *model.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("expected *model.AppError, got %T", err)
	}
}

func TestRefreshToken_UnmarshalError(t *testing.T) {
	svc, _, mr := setupServiceWithRedis(t)
	ctx := context.Background()

	// Store invalid JSON as refresh token data
	mr.Set("refresh_token:corrupt", "not-json")

	_, _, err := svc.RefreshToken(ctx, "corrupt")
	if err == nil {
		t.Fatal("expected error for corrupt refresh token data, got nil")
	}
}
