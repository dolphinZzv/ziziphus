package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewConvRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	if repo == nil {
		t.Fatal("NewConvRepo returned nil")
	}
}

func TestConvRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	c := &model.Conversation{
		ConvID:     "conv1",
		Type:       model.ConvP2P,
		Name:       "test",
		OwnerID:    "u1",
		Avatar:     "av.jpg",
		Cover:      "cv.jpg",
		MaxMembers: 2,
		CreatedAt:  time.Now().UnixMilli(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO conversations (conv_id, type, name, owner_id, avatar, cover, max_members, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`)).
		WithArgs(c.ConvID, c.Type, c.Name, c.OwnerID, c.Avatar, c.Cover, c.MaxMembers, AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), c)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestConvRepo_CreateTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectExec(`INSERT INTO conversations`).
		WithArgs("conv1", model.ConvP2P, "test", "u1", "", "", 2, AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.CreateTx(context.Background(), tx, &model.Conversation{
		ConvID: "conv1", Type: model.ConvP2P, Name: "test", OwnerID: "u1",
		MaxMembers: 2, CreatedAt: time.Now().UnixMilli(),
	})
	if err != nil {
		t.Fatalf("CreateTx: %v", err)
	}
}

func TestConvRepo_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT conv_id, type, name, owner_id, avatar, notice, max_members, last_msg_id, last_msg_at, created_at, COALESCE(settings, '{}') FROM conversations WHERE conv_id = $1`)).
		WithArgs("conv1").
		WillReturnRows(pgxmock.NewRows([]string{"conv_id", "type", "name", "owner_id", "avatar", "notice", "max_members", "last_msg_id", "last_msg_at", "created_at", "settings"}).
			AddRow("conv1", model.ConvP2P, "test", "u1", "av.jpg", "notice", 2, int64(100), &now, now, map[string]any{}))

	c, err := repo.Get(context.Background(), "conv1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c.ConvID != "conv1" {
		t.Errorf("ConvID = %q, want conv1", c.ConvID)
	}
	if c.Notice != "notice" {
		t.Errorf("Notice = %q, want notice", c.Notice)
	}
	if c.LastMsgID != 100 {
		t.Errorf("LastMsgID = %d, want 100", c.LastMsgID)
	}
}

func TestConvRepo_Get_NoLastMsg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`FROM conversations WHERE conv_id`).
		WithArgs("conv_new").
		WillReturnRows(pgxmock.NewRows([]string{"conv_id", "type", "name", "owner_id", "avatar", "notice", "max_members", "last_msg_id", "last_msg_at", "created_at", "settings"}).
			AddRow("conv_new", model.ConvP2P, "new", "u1", "", "", 2, nil, nil, now, map[string]any{}))

	c, err := repo.Get(context.Background(), "conv_new")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c.LastMsgID != 0 {
		t.Errorf("LastMsgID = %d, want 0", c.LastMsgID)
	}
	if c.LastMsgAt != 0 {
		t.Errorf("LastMsgAt = %d, want 0", c.LastMsgAt)
	}
}

func TestConvRepo_UpdateLastMsg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET last_msg_id = $1, last_msg_at = NOW() WHERE conv_id = $2`)).
		WithArgs(int64(200), "conv1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateLastMsg(context.Background(), "conv1", 200)
	if err != nil {
		t.Fatalf("UpdateLastMsg: %v", err)
	}
}

func TestConvRepo_AddMember(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO conv_members (conv_id, user_id, role, joined_at) VALUES ($1, $2, $3, NOW()) ON CONFLICT (conv_id, user_id) DO NOTHING`)).
		WithArgs("conv1", "u2", model.ConvRoleMember).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.AddMember(context.Background(), "conv1", "u2", model.ConvRoleMember)
	if err != nil {
		t.Fatalf("AddMember: %v", err)
	}
}

func TestConvRepo_AddMemberTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectExec(`INSERT INTO conv_members`).
		WithArgs("conv1", "u2", model.ConvRoleMember).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.AddMemberTx(context.Background(), tx, "conv1", "u2", model.ConvRoleMember)
	if err != nil {
		t.Fatalf("AddMemberTx: %v", err)
	}
}

func TestConvRepo_RemoveMember(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM conv_members WHERE conv_id = $1 AND user_id = $2`)).
		WithArgs("conv1", "u2").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.RemoveMember(context.Background(), "conv1", "u2")
	if err != nil {
		t.Fatalf("RemoveMember: %v", err)
	}
}

func TestConvRepo_GetMembers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT cm.conv_id, cm.user_id, cm.role, cm.nickname, cm.mute, cm.joined_at, COALESCE(u.type, 0), COALESCE(u.wake_mode, 0) FROM conv_members cm LEFT JOIN users u ON u.id = cm.user_id WHERE cm.conv_id = $1 ORDER BY cm.joined_at`)).
		WithArgs("conv1").
		WillReturnRows(pgxmock.NewRows([]string{"conv_id", "user_id", "role", "nickname", "mute", "joined_at", "user_type", "wake_mode"}).
			AddRow("conv1", "u1", model.ConvRoleOwner, "Owner", false, now, model.UserHuman, 0).
			AddRow("conv1", "u2", model.ConvRoleMember, "", false, now, model.UserHuman, 0))

	members, err := repo.GetMembers(context.Background(), "conv1")
	if err != nil {
		t.Fatalf("GetMembers: %v", err)
	}
	if len(members) != 2 {
		t.Fatalf("got %d members, want 2", len(members))
	}
	if members[0].UserID != "u1" {
		t.Errorf("members[0].UserID = %q, want u1", members[0].UserID)
	}
}

func TestConvRepo_IsMember(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM conv_members WHERE conv_id = $1 AND user_id = $2`)).
		WithArgs("conv1", "u2").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	isMember, err := repo.IsMember(context.Background(), "conv1", "u2")
	if err != nil {
		t.Fatalf("IsMember: %v", err)
	}
	if !isMember {
		t.Error("expected isMember = true")
	}
}

func TestConvRepo_GetMemberRole(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT role FROM conv_members WHERE conv_id = $1 AND user_id = $2`)).
		WithArgs("conv1", "u2").
		WillReturnRows(pgxmock.NewRows([]string{"role"}).AddRow(model.ConvRoleMember))

	role, err := repo.GetMemberRole(context.Background(), "conv1", "u2")
	if err != nil {
		t.Fatalf("GetMemberRole: %v", err)
	}
	if role != model.ConvRoleMember {
		t.Errorf("role = %d, want %d", role, model.ConvRoleMember)
	}
}

func TestConvRepo_SearchByName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM conversations WHERE type = \$1 AND name ILIKE \$2`).
		WithArgs(model.ConvGroup, "%test%").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT c.conv_id, c.name, c.avatar, c.owner_id, COALESCE(mc.count, 0), c.created_at FROM conversations c LEFT JOIN (SELECT conv_id, COUNT(*) AS count FROM conv_members GROUP BY conv_id) mc ON mc.conv_id = c.conv_id WHERE c.type = $1 AND c.name ILIKE $2 ORDER BY c.created_at DESC LIMIT $3 OFFSET $4`)).
		WithArgs(model.ConvGroup, "%test%", 10, 0).
		WillReturnRows(pgxmock.NewRows([]string{"conv_id", "name", "avatar", "owner_id", "member_count", "created_at"}).
			AddRow("g1", "Test Group", "av.jpg", "u1", 5, now))

	items, total, err := repo.SearchByName(context.Background(), "test", 1, 10)
	if err != nil {
		t.Fatalf("SearchByName: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].Name != "Test Group" {
		t.Errorf("Name = %q, want Test Group", items[0].Name)
	}
}

func TestConvRepo_UpdateNameAvatar(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET name = $1, avatar = $2 WHERE conv_id = $3`)).
		WithArgs("NewName", "new_av.jpg", "conv1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateNameAvatar(context.Background(), "conv1", "NewName", "new_av.jpg")
	if err != nil {
		t.Fatalf("UpdateNameAvatar: %v", err)
	}
}

func TestConvRepo_UpdateNotice(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET notice = $1 WHERE conv_id = $2`)).
		WithArgs("New Notice", "conv1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateNotice(context.Background(), "conv1", "New Notice")
	if err != nil {
		t.Fatalf("UpdateNotice: %v", err)
	}
}

func TestConvRepo_UpdateCover(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET cover = $1 WHERE conv_id = $2`)).
		WithArgs("new_cv.jpg", "conv1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateCover(context.Background(), "conv1", "new_cv.jpg")
	if err != nil {
		t.Fatalf("UpdateCover: %v", err)
	}
}

func TestConvRepo_Pin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(`UPDATE conv_members SET pinned = TRUE WHERE conv_id = \$1 AND user_id = \$2`).
		WithArgs("conv1", "u2").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Pin(context.Background(), "u2", "conv1")
	if err != nil {
		t.Fatalf("Pin: %v", err)
	}
}

func TestConvRepo_Unpin(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectExec(`UPDATE conv_members SET pinned = FALSE WHERE conv_id = \$1 AND user_id = \$2`).
		WithArgs("conv1", "u2").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Unpin(context.Background(), "u2", "conv1")
	if err != nil {
		t.Fatalf("Unpin: %v", err)
	}
}

func TestConvRepo_AreContacts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contacts WHERE user_id = \$1 AND contact_id = \$2`).
		WithArgs("u1", "u2").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	ok, err := repo.AreContacts(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("AreContacts: %v", err)
	}
	if !ok {
		t.Error("expected contacts = true")
	}
}

func TestConvRepo_Clone(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	id := int64(0)
	idGen := func() int64 { id++; return id }

	mock.ExpectBegin()

	mock.ExpectExec(`INSERT INTO conversations \(conv_id, type, name, owner_id, avatar, cover, notice, max_members, created_at\) SELECT \$1, type, \$2, \$3, avatar, cover, notice, max_members, NOW\(\) FROM conversations WHERE conv_id = \$4`).
		WithArgs("new_conv", "Cloned", "u1", "src_conv").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	mock.ExpectExec(`INSERT INTO conv_members \(conv_id, user_id, role, nickname, mute, pinned, joined_at\) SELECT \$1, user_id, CASE WHEN user_id = \$2 THEN 2 ELSE 0 END, nickname, mute, FALSE, NOW\(\) FROM conv_members WHERE conv_id = \$3`).
		WithArgs("new_conv", "u1", "src_conv").
		WillReturnResult(pgxmock.NewResult("INSERT", 3))

	mock.ExpectCommit()

	err = repo.Clone(context.Background(), "src_conv", "new_conv", "u1", "Cloned", idGen)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestConvRepo_GetUserConvs(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)
	now := time.Now()

	// Count query
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM conv_members WHERE user_id = \$1`).
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	// GetUserConvs query
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT c.conv_id, c.type, c.name, c.avatar, COALESCE(m.msg_id, 0), COALESCE(m.sender_id, ''), COALESCE(u.name, ''), COALESCE(m.body, ''), COALESCE(m.content_type, 0), COALESCE(m.timestamp, 0), COALESCE(m.status, 0), c.last_msg_at, cm.role, cm.mute, cm.pinned FROM conv_members cm JOIN conversations c ON c.conv_id = cm.conv_id LEFT JOIN messages m ON m.msg_id = c.last_msg_id LEFT JOIN users u ON u.id = m.sender_id WHERE cm.user_id = $1 ORDER BY cm.pinned DESC, c.last_msg_at DESC NULLS LAST LIMIT $2 OFFSET $3`)).
		WithArgs("u1", 20, 0).
		WillReturnRows(pgxmock.NewRows([]string{
			"conv_id", "type", "name", "avatar",
			"last_msg_id", "last_sender_id", "last_sender_name", "last_body",
			"last_content_type", "last_timestamp", "last_status",
			"last_msg_at", "role", "mute", "pinned",
		}).
			AddRow("g1", model.ConvGroup, "Group Chat", "av.jpg",
				int64(100), "u2", "Bob", "Hello!", 1, int64(1000), int64(1),
				&now, model.ConvRoleOwner, false, true).
			AddRow("u1:u3", model.ConvP2P, "u1:u3", "",
				0, "", "", "", 0, int64(0), 0,
				nil, model.ConvRoleMember, false, false))

	// Resolve partner names - for u1:u3 P2P
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT u.id, u.name, u.type, COALESCE(c.nickname, '') FROM users u LEFT JOIN contacts c ON c.user_id = $1 AND c.contact_id = u.id WHERE u.id = ANY($2)`)).
		WithArgs("u1", []string{"u3"}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "nickname"}).
			AddRow("u3", "Charlie", model.UserHuman, ""))

	items, total, err := repo.GetUserConvs(context.Background(), "u1", 1, 20)
	if err != nil {
		t.Fatalf("GetUserConvs: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	if items[0].Name != "Group Chat" {
		t.Errorf("items[0].Name = %q, want Group Chat", items[0].Name)
	}
	if items[1].Name != "Charlie" {
		t.Errorf("items[1].Name = %q, want Charlie (resolved)", items[1].Name)
	}
	if items[1].PartnerType != int(model.UserHuman) {
		t.Errorf("items[1].PartnerType = %d, want %d", items[1].PartnerType, model.UserHuman)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestConvRepo_GetUserConvs_NoLastMsg(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewConvRepo(mock)

	mock.ExpectQuery(`COUNT\(\*\) FROM conv_members`).
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`SELECT c.conv_id, c.type, c.name, c.avatar`).
		WithArgs("u1", 20, 0).
		WillReturnRows(pgxmock.NewRows([]string{
			"conv_id", "type", "name", "avatar",
			"last_msg_id", "last_sender_id", "last_sender_name", "last_body",
			"last_content_type", "last_timestamp", "last_status",
			"last_msg_at", "role", "mute", "pinned",
		}).AddRow("p2p_conv", model.ConvP2P, "p2p_conv", "",
			0, "", "", "", 0, int64(0), 0,
			nil, model.ConvRoleMember, false, false))

	mock.ExpectQuery(`SELECT u.id, u.name, u.type, COALESCE\(c.nickname, ''\)`).
		WithArgs("u1", []string{"p2p_conv"}).
		WillReturnRows(pgxmock.NewRows([]string{"id", "name", "type", "nickname"}).
			AddRow("p2p_conv", "", model.UserHuman, ""))

	items, _, err := repo.GetUserConvs(context.Background(), "u1", 1, 20)
	if err != nil {
		t.Fatalf("GetUserConvs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("got %d items, want 1", len(items))
	}
	if items[0].LastMessage != nil {
		t.Error("expected LastMessage to be nil for conv without messages")
	}
}
