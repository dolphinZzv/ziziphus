package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
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

func userTestUser() *model.User {
	return &model.User{
		ID:        "u1",
		Type:      model.UserHuman,
		Name:      "Alice",
		Email:     "",
		Avatar:    "avatar.jpg",
		Cover:     "cover.jpg",
		Status:    1,
		Banned:    false,
		Password:  "hashed",
		ExtMeta:   map[string]any{"key": "val"},
		CreatedAt: time.Now().UnixMilli(),
		Account:   "alice",
		Language:  "zh-Hans",
	}
}

// columnOrder returns the common scan/column order for SELECT queries on users.
func userColumnOrder() []string {
	return []string{
		"id", "type", "name", "email", "avatar", "cover", "status",
		"banned",
		"created_at", "account", "primary_color", "secondary_color",
		"uid", "wake_mode", "api_key", "discoverable", "allow_direct_chat", "headline", "language",
		"conv_limit",
	}
}

func TestUserRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	u := userTestUser()

	cols := `INSERT INTO users \(id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language\) VALUES \(\$1, \$2, \$3, \$4, \$5, \$6, \$7, \$8, \$9, \$10, \$11, \$12, \$13, \$14, \$15, \$16, \$17, \$18, \$19, \$20, \$21\)`
	mock.ExpectExec(cols).
		WithArgs(u.ID, u.Type, u.Name, u.Email, u.Avatar, u.Cover, u.Status, u.Banned,
			u.Password, u.ExtMeta, AnyTime{}, u.Account,
			u.PrimaryColor, u.SecondaryColor, u.UID, u.WakeMode,
			u.APIKey, u.Discoverable, u.AllowDirectChat, u.Headline, u.Language).
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
	u := userTestUser()

	mock.ExpectExec(`INSERT INTO users`).
		WithArgs(u.ID, u.Type, u.Name, u.Email, u.Avatar, u.Cover, u.Status, u.Banned,
			u.Password, u.ExtMeta, AnyTime{}, u.Account,
			u.PrimaryColor, u.SecondaryColor, u.UID, u.WakeMode,
			u.APIKey, u.Discoverable, u.AllowDirectChat, u.Headline, u.Language).
		WillReturnError(context.DeadlineExceeded)

	err = repo.Create(context.Background(), u)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func fullUserCols() []string {
	return []string{
		"id", "type", "name", "email", "avatar", "cover", "status",
		"banned",
		"password", "ext_meta",
		"created_at", "account", "primary_color", "secondary_color",
		"uid", "wake_mode", "api_key", "discoverable", "allow_direct_chat", "headline", "language",
		"conv_limit",
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

	rows := pgxmock.NewRows(fullUserCols()).
		AddRow("u1", model.UserHuman, "Alice", "", "av.jpg", "", 1,
			false,
			"pwd_hash", map[string]any{},
			now, "alice", "#f00", "#0f0",
			"", 0, "", true, true, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE id = $1`)).
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
	if user.Discoverable != true {
		t.Errorf("Discoverable = %v, want true", user.Discoverable)
	}
	if user.AllowDirectChat != true {
		t.Errorf("AllowDirectChat = %v, want true", user.AllowDirectChat)
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

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE id = $1`)).
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

	rows := pgxmock.NewRows(userColumnOrder()).
		AddRow("u1", model.UserHuman, "Alice", "", "av1.jpg", "cover1", 1,
			false, now, "alice", "", "",
			"", 0, "", true, false, "", "zh-Hans", 100).
		AddRow("u2", model.UserHuman, "Bob", "", "av2.jpg", "cover2", 1,
			false, now, "bob", "", "",
			"", 0, "", false, true, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE id = ANY($1)`)).
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
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE name ILIKE \$1 AND NOT banned`).
		WithArgs("%ali%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	// Search query
	rows := pgxmock.NewRows(userColumnOrder()).
		AddRow("u1", model.UserHuman, "Alice", "", "av.jpg", "cv.jpg", 1,
			false, now, "alice", "", "",
			"", 0, "", false, false, "", "zh-Hans", 100).
		AddRow("u3", model.UserHuman, "Alicia", "", "av3.jpg", "cv3.jpg", 1,
			false, now, "alicia", "", "",
			"", 0, "", false, false, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE name ILIKE $1 AND NOT banned ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
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

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE name ILIKE \$1 AND NOT banned`).
		WithArgs("%zzz%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	rows := pgxmock.NewRows(userColumnOrder())
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE name ILIKE $1 AND NOT banned ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
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

	rows := pgxmock.NewRows(fullUserCols()).
		AddRow("u1", model.UserHuman, "Alice", "", "av.jpg", "cv.jpg", 1,
			false,
			"pwd", map[string]any{},
			now, "alice", "", "",
			"", 0, "", false, false, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE account = $1`)).
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

	mock.ExpectExec(`UPDATE users SET name = \$1, avatar = \$2, cover = \$3, email = \$4, primary_color = \$5, secondary_color = \$6, discoverable = \$7, allow_direct_chat = \$8, headline = \$9 WHERE id = \$10`).
		WithArgs("NewName", "new_av.jpg", "cover.jpg", "test@email.com", "#FF0000", "#00FF00", true, true, "", "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Update(context.Background(), "u1", "NewName", "new_av.jpg", "cover.jpg", "test@email.com", "#FF0000", "#00FF00", "", true, true)
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

// ---------------------------------------------------------------------------
// New tests for untested methods
// ---------------------------------------------------------------------------

func TestUserRepo_CountAgents(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM users WHERE type = \$1 AND uid = \$2`).
		WithArgs(model.UserAgent, "owner_1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(3))

	count, err := repo.CountAgents(context.Background(), "owner_1")
	if err != nil {
		t.Fatalf("CountAgents: %v", err)
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_ListAgents(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows(userColumnOrder()).
		AddRow("agent_1", model.UserAgent, "Bot1", "", "", "", 0,
			false, now, "", "", "",
			"owner_1", 0, "", false, false, "", "zh-Hans", 100).
		AddRow("agent_2", model.UserAgent, "Bot2", "", "a.jpg", "c.jpg", 0,
			false, now, "", "", "",
			"owner_1", 0, "key_2", false, false, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE type = $1 AND uid = $2 ORDER BY created_at ASC`)).
		WithArgs(model.UserAgent, "owner_1").
		WillReturnRows(rows)

	agents, err := repo.ListAgents(context.Background(), "owner_1")
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("got %d agents, want 2", len(agents))
	}
	if agents[0].Name != "Bot1" {
		t.Errorf("agents[0].Name = %q, want Bot1", agents[0].Name)
	}
	if agents[1].APIKey != "key_2" {
		t.Errorf("agents[1].APIKey = %q, want key_2", agents[1].APIKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_UpdateAgent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(`UPDATE users SET name = \$1, avatar = \$2, cover = \$3, primary_color = \$4, secondary_color = \$5, wake_mode = \$6, discoverable = \$7, allow_direct_chat = \$8, headline = \$9 WHERE id = \$10 AND type = \$11 AND uid = \$12`).
		WithArgs("AgentX", "av.png", "cv.png", "#111", "#222", model.WakeModeAll, true, false, "", "agent_1", model.UserAgent, "owner_1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateAgent(context.Background(), "agent_1", "owner_1", "AgentX", "av.png", "cv.png", "#111", "#222", "", model.WakeModeAll, true, false)
	if err != nil {
		t.Fatalf("UpdateAgent: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_DeleteAgent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(`DELETE FROM users WHERE id = \$1 AND type = \$2 AND uid = \$3`).
		WithArgs("agent_1", model.UserAgent, "owner_1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.DeleteAgent(context.Background(), "agent_1", "owner_1")
	if err != nil {
		t.Fatalf("DeleteAgent: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_GetByAPIKey(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows(userColumnOrder()).
		AddRow("agent_1", model.UserAgent, "Bot", "", "", "", 0,
			false, now, "", "", "",
			"owner_1", 0, "sk-test-123", false, false, "", "zh-Hans", 100)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE api_key = $1`)).
		WithArgs("sk-test-123").
		WillReturnRows(rows)

	user, err := repo.GetByAPIKey(context.Background(), "sk-test-123")
	if err != nil {
		t.Fatalf("GetByAPIKey: %v", err)
	}
	if user.APIKey != "sk-test-123" {
		t.Errorf("APIKey = %q, want sk-test-123", user.APIKey)
	}
	if user.UID != "owner_1" {
		t.Errorf("UID = %q, want owner_1", user.UID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_GetByAPIKey_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language, COALESCE(conv_limit, 100) FROM users WHERE api_key = $1`)).
		WithArgs("invalid-key").
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.GetByAPIKey(context.Background(), "invalid-key")
	if err == nil {
		t.Fatal("expected error for invalid api key")
	}
}

func TestUserRepo_UpdateAgentAPIKey(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(`UPDATE users SET api_key = \$1 WHERE id = \$2 AND type = \$3 AND uid = \$4`).
		WithArgs("new-key", "agent_1", model.UserAgent, "owner_1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateAgentAPIKey(context.Background(), "agent_1", "owner_1", "new-key")
	if err != nil {
		t.Fatalf("UpdateAgentAPIKey: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_DeleteAccount(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)
	uid := "user_to_delete"

	// Expect a transaction
	mock.ExpectBegin()

	// 1. Delete conv members
	mock.ExpectExec(`DELETE FROM conv_members WHERE user_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 2))

	// 2. Anonymize messages
	mock.ExpectExec(`UPDATE messages SET body = '' WHERE sender_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("UPDATE", 5))

	// 3. Delete join requests
	mock.ExpectExec(`DELETE FROM join_requests WHERE user_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// 4. Delete contacts (both directions)
	mock.ExpectExec(`DELETE FROM contacts WHERE user_id = \$1 OR contact_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 3))

	// 5. Delete contact requests
	mock.ExpectExec(`DELETE FROM contact_requests WHERE from_user_id = \$1 OR to_user_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 2))

	// 6. Delete msg receipts
	mock.ExpectExec(`DELETE FROM msg_receipts WHERE user_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 0))

	// 7. Clear owner references
	mock.ExpectExec(`UPDATE conversations SET owner_id = '' WHERE owner_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// 8. Delete sessions
	mock.ExpectExec(`DELETE FROM sessions WHERE user_id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	// 9. Delete the user
	mock.ExpectExec(`DELETE FROM users WHERE id = \$1`).
		WithArgs(uid).WillReturnResult(pgxmock.NewResult("DELETE", 1))

	mock.ExpectCommit()

	err = repo.DeleteAccount(context.Background(), uid)
	if err != nil {
		t.Fatalf("DeleteAccount: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_BanUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET banned = true WHERE id = $1`)).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.BanUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("BanUser: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_UnbanUser(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET banned = false WHERE id = $1`)).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UnbanUser(context.Background(), "u1")
	if err != nil {
		t.Fatalf("UnbanUser: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_UpdateLanguage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET language = $1 WHERE id = $2`)).
		WithArgs("zh-CN", "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateLanguage(context.Background(), "u1", "zh-CN")
	if err != nil {
		t.Fatalf("UpdateLanguage: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestUserRepo_IsBanned(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	t.Run("banned user returns true", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT banned FROM users WHERE id = $1`)).
			WithArgs("u_banned").
			WillReturnRows(pgxmock.NewRows([]string{"banned"}).AddRow(true))

		banned, err := repo.IsBanned(context.Background(), "u_banned")
		if err != nil {
			t.Fatalf("IsBanned: %v", err)
		}
		if !banned {
			t.Error("IsBanned = false, want true")
		}
	})

	t.Run("not banned user returns false", func(t *testing.T) {
		mock.ExpectQuery(regexp.QuoteMeta(`SELECT banned FROM users WHERE id = $1`)).
			WithArgs("u_active").
			WillReturnRows(pgxmock.NewRows([]string{"banned"}).AddRow(false))

		banned, err := repo.IsBanned(context.Background(), "u_active")
		if err != nil {
			t.Fatalf("IsBanned: %v", err)
		}
		if banned {
			t.Error("IsBanned = true, want false")
		}
	})
}

func TestUserRepo_UpdateBanned(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewUserRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE users SET banned = $1 WHERE id = $2`)).
		WithArgs(true, "u1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateBanned(context.Background(), "u1", true)
	if err != nil {
		t.Fatalf("UpdateBanned: %v", err)
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
