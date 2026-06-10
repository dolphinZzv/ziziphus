package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewUserRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	if repo == nil {
		t.Fatal("NewUserRepo returned nil")
	}
}

func TestUserRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	u := &model.User{
		ID:        "u1",
		Account:   "alice",
		Name:      "Alice",
		Avatar:    "avatar.jpg",
		Status:    1,
		Password:  "hashed",
		ExtMeta:   map[string]any{"key": "val"},
		CreatedAt: time.Now().UnixMilli(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO users (id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`)).
		WithArgs(u.ID, u.Type, u.Name, u.Avatar, u.Status, u.Password, u.ExtMeta, AnyTime{}, u.Account, u.PrimaryColor, u.SecondaryColor).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), u)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_Create_Error(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	u := &model.User{ID: "u1", Account: "alice", Name: "Alice", CreatedAt: 1000}

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(u.ID, u.Type, u.Name, u.Avatar, u.Status, u.Password, u.ExtMeta, AnyTime{}, u.Account, u.PrimaryColor, u.SecondaryColor).
		WillReturnError(context.DeadlineExceeded)

	err = repo.Create(context.Background(), u)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestUserRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "type", "name", "avatar", "status", "password", "ext_meta", "created_at", "account", "primary_color", "secondary_color"}).
		AddRow("u1", 0, "Alice", "av.jpg", 1, "pwd", map[string]any{}, now, "alice", "", "")

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color FROM users WHERE id = $1`)).
		WithArgs("u1").
		WillReturnRows(rows)

	user, err := repo.GetByID(context.Background(), "u1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("ID = %q, want u1", user.ID)
	}
	if user.Name != "Alice" {
		t.Errorf("Name = %q, want Alice", user.Name)
	}
	if user.Account != "alice" {
		t.Errorf("Account = %q, want alice", user.Account)
	}
	if user.CreatedAt != now.UnixMilli() {
		t.Errorf("CreatedAt = %d, want %d", user.CreatedAt, now.UnixMilli())
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectQuery(`SELECT id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color FROM users WHERE id = \$1`).
		WithArgs("nonexistent").
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.GetByID(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestUserRepo_GetByIDs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "type", "name", "avatar", "status", "created_at", "account", "primary_color", "secondary_color"}).
		AddRow("u1", 0, "Alice", "av1.jpg", 1, now, "alice", "", "").
		AddRow("u2", 0, "Bob", "av2.jpg", 1, now, "bob", "", "")

	mock.ExpectQuery(`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color FROM users WHERE id = ANY\(\$1\)`).
		WithArgs([]string{"u1", "u2"}).
		WillReturnRows(rows)

	users, err := repo.GetByIDs(context.Background(), []string{"u1", "u2"})
	if err != nil {
		t.Fatalf("GetByIDs: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}
	if users["u1"].Name != "Alice" {
		t.Errorf("u1.Name = %q, want Alice", users["u1"].Name)
	}
	if users["u2"].Name != "Bob" {
		t.Errorf("u2.Name = %q, want Bob", users["u2"].Name)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_Search(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	// Count query
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE name ILIKE \$1`).
		WithArgs("%ali%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	// Search query
	rows := pgxmock.NewRows([]string{"id", "type", "name", "avatar", "status", "created_at", "account", "primary_color", "secondary_color"}).
		AddRow("u1", 0, "Alice", "av.jpg", 1, now, "alice", "", "").
		AddRow("u3", 0, "Alicia", "av3.jpg", 1, now, "alicia", "", "")

	mock.ExpectQuery(`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color FROM users WHERE name ILIKE \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs("%ali%", 10, 0).
		WillReturnRows(rows)

	users, total, err := repo.Search(context.Background(), "ali", 1, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(users) != 2 {
		t.Fatalf("got %d users, want 2", len(users))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_Search_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE name ILIKE \$1`).
		WithArgs("%zzz%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	rows := pgxmock.NewRows([]string{"id", "type", "name", "avatar", "status", "created_at", "account", "primary_color", "secondary_color"})
	mock.ExpectQuery(`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color FROM users WHERE name ILIKE \$1 ORDER BY created_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs("%zzz%", 10, 0).
		WillReturnRows(rows)

	users, total, err := repo.Search(context.Background(), "zzz", 1, 10)
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(users) != 0 {
		t.Errorf("got %d users, want 0", len(users))
	}
}

func TestUserRepo_GetByAccount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"id", "type", "name", "avatar", "status", "password", "ext_meta", "created_at", "account", "primary_color", "secondary_color"}).
		AddRow("u1", 0, "Alice", "av.jpg", 1, "pwd", map[string]any{}, now, "alice", "", "")

	mock.ExpectQuery(`SELECT id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color FROM users WHERE account = \$1`).
		WithArgs("alice").
		WillReturnRows(rows)

	user, err := repo.GetByAccount(context.Background(), "alice")
	if err != nil {
		t.Fatalf("GetByAccount: %v", err)
	}
	if user.ID != "u1" {
		t.Errorf("ID = %q, want u1", user.ID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_Update(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(`UPDATE users SET name = \$1, avatar = \$2, primary_color = \$3, secondary_color = \$4 WHERE id = \$5`).
		WithArgs("NewName", "new_av.jpg", "", "", "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), "u1", "NewName", "new_av.jpg", "", "")
	if err != nil {
		t.Fatalf("Update: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

// AnyTime is a custom matcher for time.Time values in pgxmock.
type AnyTime struct{}

func (a AnyTime) Match(v any) bool {
	_, ok := v.(time.Time)
	return ok
}
