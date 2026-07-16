package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
)

func TestNewContactRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)
	if repo == nil {
		t.Fatal("NewContactRepo returned nil")
	}
}

func TestContactRepo_Add(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)
	c := &model.Contact{
		UserID:    "u1",
		ContactID: "u2",
		Nickname:  "Buddy",
		AddedAt:   time.Now().UnixMilli(),
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO contacts (user_id, contact_id, nickname, added_at) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, contact_id) DO NOTHING`)).
		WithArgs(c.UserID, c.ContactID, c.Nickname, AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Add(context.Background(), c)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRepo_AddContact(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO contacts (user_id, contact_id, nickname, added_at) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, contact_id) DO NOTHING`)).
		WithArgs("u1", "u2", "", AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.AddContact(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("AddContact: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRepo_Remove(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM contacts WHERE user_id = $1 AND contact_id = $2`)).
		WithArgs("u1", "u2").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Remove(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRepo_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)
	now := time.Now()

	// Count
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM contacts WHERE user_id = $1`)).
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(2))

	// List
	rows := pgxmock.NewRows([]string{"user_id", "contact_id", "nickname", "added_at"}).
		AddRow("u1", "u2", "Buddy", now).
		AddRow("u1", "u3", "Friend", now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT user_id, contact_id, nickname, added_at FROM contacts WHERE user_id = $1 ORDER BY added_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs("u1", 10, 0).
		WillReturnRows(rows)

	contacts, total, err := repo.List(context.Background(), "u1", 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(contacts) != 2 {
		t.Fatalf("got %d contacts, want 2", len(contacts))
	}
	if contacts[0].ContactID != "u2" {
		t.Errorf("contacts[0].ContactID = %q, want u2", contacts[0].ContactID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRepo_List_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contacts WHERE user_id = \$1`).
		WithArgs("u_empty").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	rows := pgxmock.NewRows([]string{"user_id", "contact_id", "nickname", "added_at"})
	mock.ExpectQuery(`SELECT user_id, contact_id, nickname, added_at FROM contacts WHERE user_id = \$1 ORDER BY added_at DESC LIMIT \$2 OFFSET \$3`).
		WithArgs("u_empty", 10, 0).
		WillReturnRows(rows)

	contacts, total, err := repo.List(context.Background(), "u_empty", 1, 10)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
	if len(contacts) != 0 {
		t.Errorf("got %d contacts, want 0", len(contacts))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRepo_UpdateNickname(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE contacts SET nickname = $1 WHERE user_id = $2 AND contact_id = $3`)).
		WithArgs("NewNick", "u1", "u2").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateNickname(context.Background(), "u1", "u2", "NewNick")
	if err != nil {
		t.Fatalf("UpdateNickname: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
