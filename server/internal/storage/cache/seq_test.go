package cache

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupSeqCache(t *testing.T) (*miniredis.Miniredis, *SeqCache) {
	t.Helper()
	mr := miniredis.RunT(t)
	client, err := NewRedisClient(mr.Addr(), "", 0)
	if err != nil {
		t.Fatalf("NewRedisClient: %v", err)
	}
	return mr, NewSeqCache(client)
}

func TestNewSeqCache(t *testing.T) {
	_, sc := setupSeqCache(t)
	if sc == nil {
		t.Fatal("NewSeqCache returned nil")
	}
}

func TestGetAndIncrementConvSeq(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	seq, err := sc.GetAndIncrementConvSeq(ctx, "conv_1")
	if err != nil {
		t.Fatalf("first GetAndIncrementConvSeq: %v", err)
	}
	if seq != 1 {
		t.Errorf("seq = %d, want 1", seq)
	}

	seq, err = sc.GetAndIncrementConvSeq(ctx, "conv_1")
	if err != nil {
		t.Fatalf("second GetAndIncrementConvSeq: %v", err)
	}
	if seq != 2 {
		t.Errorf("seq = %d, want 2", seq)
	}
}

func TestGetAndIncrementConvSeq_DifferentKeys(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	s1, _ := sc.GetAndIncrementConvSeq(ctx, "conv_a")
	s2, _ := sc.GetAndIncrementConvSeq(ctx, "conv_b")
	if s1 != 1 || s2 != 1 {
		t.Errorf("s1=%d s2=%d, want both 1", s1, s2)
	}
}

func TestSetGetUserSeq(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	err := sc.SetUserSeq(ctx, "user_1", "conv_1", 42)
	if err != nil {
		t.Fatalf("SetUserSeq: %v", err)
	}

	seq, err := sc.GetUserSeq(ctx, "user_1", "conv_1")
	if err != nil {
		t.Fatalf("GetUserSeq: %v", err)
	}
	if seq != 42 {
		t.Errorf("seq = %d, want 42", seq)
	}
}

func TestGetUserSeq_NotExists(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	_, err := sc.GetUserSeq(ctx, "nonexistent", "conv_1")
	if err != redis.Nil {
		t.Errorf("expected redis.Nil, got %v", err)
	}
}

func TestSetGetSessionSeq(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	err := sc.SetSessionSeq(ctx, "session_1", "conv_1", 77)
	if err != nil {
		t.Fatalf("SetSessionSeq: %v", err)
	}

	seq, err := sc.GetSessionSeq(ctx, "session_1", "conv_1")
	if err != nil {
		t.Fatalf("GetSessionSeq: %v", err)
	}
	if seq != 77 {
		t.Errorf("seq = %d, want 77", seq)
	}
}

func TestMarkRead_ReturnsUnreadCount(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	// Set conv seq to 10
	mr.Set("conv:seq:conv_x", "10")

	unread, err := sc.MarkRead(ctx, "user_1", "conv_x", 7)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if unread != 3 {
		t.Errorf("unread = %d, want 3 (10-7)", unread)
	}
}

func TestMarkRead_ClampsNegatives(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_x", "5")

	unread, err := sc.MarkRead(ctx, "user_1", "conv_x", 10)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if unread != 0 {
		t.Errorf("unread = %d, want 0 (clamped)", unread)
	}
}

func TestMarkRead_NoConvSeq(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	// Set conv seq to 0 so the pipeline GET returns a value
	mr.Set("conv:seq:conv_no_seq", "0")

	unread, err := sc.MarkRead(ctx, "u1", "conv_no_seq", 3)
	if err != nil {
		t.Fatalf("MarkRead: %v", err)
	}
	if unread != 0 {
		t.Errorf("unread = %d, want 0 (clamped negative)", unread)
	}
}

func TestGetUnreadCount(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_x", "100")
	mr.Set("user:seq:u1:conv_x", "80")

	count, err := sc.GetUnreadCount(ctx, "u1", "conv_x")
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 20 {
		t.Errorf("unread = %d, want 20", count)
	}
}

func TestGetUnreadCount_NoUserSeq(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_x", "50")

	count, err := sc.GetUnreadCount(ctx, "u1", "conv_x")
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 50 {
		t.Errorf("unread = %d, want 50 (user seq defaults to 0)", count)
	}
}

func TestGetUnreadCount_NoConvSeq(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	count, err := sc.GetUnreadCount(ctx, "u1", "conv_noexist")
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 0 {
		t.Errorf("unread = %d, want 0", count)
	}
}

func TestGetUnreadCount_NoOverflow(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_x", "5")
	mr.Set("user:seq:u1:conv_x", "10")

	count, err := sc.GetUnreadCount(ctx, "u1", "conv_x")
	if err != nil {
		t.Fatalf("GetUnreadCount: %v", err)
	}
	if count != 0 {
		t.Errorf("unread = %d, want 0 (clamped)", count)
	}
}

func TestSetRecentMsg(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	err := sc.SetRecentMsg(ctx, "conv_x", 100, 1000.0)
	if err != nil {
		t.Fatalf("SetRecentMsg: %v", err)
	}

	err = sc.SetRecentMsg(ctx, "conv_x", 101, 1001.0)
	if err != nil {
		t.Fatalf("SetRecentMsg (2nd): %v", err)
	}
}

func TestInitConvSeq(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	err := sc.InitConvSeq(ctx, "conv_new", 50)
	if err != nil {
		t.Fatalf("InitConvSeq: %v", err)
	}

	seq, err := sc.GetAndIncrementConvSeq(ctx, "conv_new")
	if err != nil {
		t.Fatalf("GetAndIncrementConvSeq after Init: %v", err)
	}
	if seq != 51 {
		t.Errorf("seq = %d, want 51 (initial 50 + 1)", seq)
	}
}

func TestInitConvSeq_AlreadyExists(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_existing", "30")

	// Init with lower value - should not override
	err := sc.InitConvSeq(ctx, "conv_existing", 20)
	if err != nil {
		t.Fatalf("InitConvSeq (lower): %v", err)
	}

	seq, _ := sc.GetAndIncrementConvSeq(ctx, "conv_existing")
	if seq != 31 {
		t.Errorf("seq = %d, want 31 (existing 30 + 1)", seq)
	}
}

func TestInitConvSeq_UpdateHigher(t *testing.T) {
	mr, sc := setupSeqCache(t)
	ctx := context.Background()

	mr.Set("conv:seq:conv_existing", "10")

	// Init with higher value - should update via Lua
	err := sc.InitConvSeq(ctx, "conv_existing", 50)
	if err != nil {
		t.Fatalf("InitConvSeq (higher): %v", err)
	}

	seq, _ := sc.GetAndIncrementConvSeq(ctx, "conv_existing")
	if seq != 51 {
		t.Errorf("seq = %d, want 51 (updated to 50 + 1)", seq)
	}
}

func TestRecoverConvSeq_DelegatesToInit(t *testing.T) {
	_, sc := setupSeqCache(t)
	ctx := context.Background()

	err := sc.RecoverConvSeq(ctx, "conv_recover", 100)
	if err != nil {
		t.Fatalf("RecoverConvSeq: %v", err)
	}

	seq, _ := sc.GetAndIncrementConvSeq(ctx, "conv_recover")
	if seq != 101 {
		t.Errorf("seq = %d, want 101", seq)
	}
}
