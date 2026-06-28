package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewContactRequestRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	if repo == nil {
		t.Fatal("NewContactRequestRepo returned nil")
	}
}

func TestContactRequestRepo_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	req := &model.ContactRequest{
		FromUserID: "u1",
		ToUserID:   "u2",
		FormMsgID:  0,
		Status:     model.ContactRequestPending,
		Message:    "hello",
	}

	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO contact_requests (from_user_id, to_user_id, form_msg_id, status, message) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (from_user_id, to_user_id) DO NOTHING RETURNING id`)).
		WithArgs(req.FromUserID, req.ToUserID, req.FormMsgID, req.Status, req.Message).
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(42)))

	id, err := repo.Insert(context.Background(), req)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if id != 42 {
		t.Errorf("id = %d, want 42", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestContactRequestRepo_Insert_Conflict(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectQuery(`INSERT INTO contact_requests`).
		WithArgs("u1", "u2", int64(0), model.ContactRequestPending, "hello").
		WillReturnError(pgx.ErrNoRows)

	id, err := repo.Insert(context.Background(), &model.ContactRequest{
		FromUserID: "u1", ToUserID: "u2", Status: model.ContactRequestPending, Message: "hello",
	})
	if err != nil {
		t.Fatalf("Insert conflict: %v", err)
	}
	if id != 0 {
		t.Errorf("id = %d, want 0", id)
	}
}

func TestContactRequestRepo_Insert_GenericError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectQuery(`INSERT INTO contact_requests`).
		WithArgs("u1", "u2", int64(0), model.ContactRequestPending, "hello").
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.Insert(context.Background(), &model.ContactRequest{
		FromUserID: "u1", ToUserID: "u2", Status: model.ContactRequestPending, Message: "hello",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestContactRequestRepo_InsertTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectQuery(`INSERT INTO contact_requests`).
		WithArgs("u1", "u2", int64(0), model.ContactRequestPending, "hello").
		WillReturnRows(pgxmock.NewRows([]string{"id"}).AddRow(int64(42)))

		id, err := repo.InsertTx(context.Background(), tx, &model.ContactRequest{
			FromUserID: "u1", ToUserID: "u2", Status: model.ContactRequestPending, Message: "hello",
		})
		if err != nil {
			t.Fatalf("InsertTx: %v", err)
	}
	if id != 42 {
		t.Errorf("id = %d, want 42", id)
	}
}

func TestContactRequestRepo_UpdateFormMsgID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE contact_requests SET form_msg_id = $1, updated_at = NOW() WHERE id = $2`)).
		WithArgs(int64(100), int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateFormMsgID(context.Background(), 42, 100)
	if err != nil {
		t.Fatalf("UpdateFormMsgID: %v", err)
	}
}

func TestContactRequestRepo_UpdateFormMsgIDTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectExec(`UPDATE contact_requests SET form_msg_id`).
		WithArgs(int64(100), int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateFormMsgIDTx(context.Background(), tx, 42, 100)
	if err != nil {
		t.Fatalf("UpdateFormMsgIDTx: %v", err)
	}
}

func TestContactRequestRepo_UpdateStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE contact_requests SET status = $1, updated_at = NOW() WHERE id = $2`)).
		WithArgs(model.ContactRequestApproved, int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateStatus(context.Background(), 42, model.ContactRequestApproved)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}
}

func TestContactRequestRepo_UpdateStatusTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectExec(`UPDATE contact_requests SET status`).
		WithArgs(model.ContactRequestApproved, int64(42)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateStatusTx(context.Background(), tx, 42, model.ContactRequestApproved)
	if err != nil {
		t.Fatalf("UpdateStatusTx: %v", err)
	}
}

func contactRequestRow(id int64, from, to string, status int, now time.Time) []any {
	return []any{id, from, to, int64(formMsgID), status, inMsg,
		now, now}
}

var formMsgID int64 = 10
var inMsg = "hello"

func contactRequestColumns() []string {
	return []string{"id", "from_user_id", "to_user_id", "form_msg_id", "status", "message", "created_at", "updated_at"}
}

func TestContactRequestRepo_LockByIDTx(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	now := time.Now()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE id = $1 FOR UPDATE`)).
		WithArgs(int64(42)).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(42), "u1", "u2", int64(10), model.ContactRequestPending, "hello", now, now))

	req, err := repo.LockByIDTx(context.Background(), tx, 42)
	if err != nil {
		t.Fatalf("LockByIDTx: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if req.ID != 42 {
		t.Errorf("ID = %d, want 42", req.ID)
	}
}

func TestContactRequestRepo_LockByIDTx_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	mock.ExpectBegin()
	tx, err := mock.Begin(context.Background())
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}

	mock.ExpectQuery(`FOR UPDATE`).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	req, err := repo.LockByIDTx(context.Background(), tx, 999)
	if err != nil {
		t.Fatalf("LockByIDTx: %v", err)
	}
	if req != nil {
		t.Fatal("expected nil for not found")
	}
}

func TestContactRequestRepo_GetByID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE id = $1`)).
		WithArgs(int64(42)).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(42), "u1", "u2", int64(10), model.ContactRequestPending, "hello", now, now))

	req, err := repo.GetByID(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if req.FromUserID != "u1" {
		t.Errorf("FromUserID = %q, want u1", req.FromUserID)
	}
}

func TestContactRequestRepo_GetByID_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectQuery(`FROM contact_requests WHERE id`).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	req, err := repo.GetByID(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if req != nil {
		t.Fatal("expected nil for not found")
	}
}

func TestContactRequestRepo_GetByFormMsgID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE form_msg_id = $1`)).
		WithArgs(int64(100)).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(1), "u1", "u2", int64(100), model.ContactRequestPending, "hi", now, now))

	req, err := repo.GetByFormMsgID(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetByFormMsgID: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
	if req.FormMsgID != 100 {
		t.Errorf("FormMsgID = %d, want 100", req.FormMsgID)
	}
}

func TestContactRequestRepo_GetByPair(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE from_user_id = $1 AND to_user_id = $2`)).
		WithArgs("u1", "u2").
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(1), "u1", "u2", int64(0), model.ContactRequestPending, "", now, now))

	req, err := repo.GetByPair(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("GetByPair: %v", err)
	}
	if req == nil {
		t.Fatal("expected non-nil request")
	}
}

func TestContactRequestRepo_ListSent(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contact_requests WHERE from_user_id = \$1`).
		WithArgs("u1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE from_user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs("u1", 10, 0).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(1), "u1", "u2", int64(0), model.ContactRequestPending, "hi", now, now))

	results, total, err := repo.ListSent(context.Background(), "u1", 1, 10)
	if err != nil {
		t.Fatalf("ListSent: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestContactRequestRepo_ListReceived(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contact_requests WHERE to_user_id = \$1`).
		WithArgs("u2").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, from_user_id, to_user_id, form_msg_id, status, message, created_at, updated_at FROM contact_requests WHERE to_user_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`)).
		WithArgs("u2", 10, 0).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(1), "u1", "u2", int64(0), model.ContactRequestPending, "hi", now, now))

	results, total, err := repo.ListReceived(context.Background(), "u2", -1, 1, 10)
	if err != nil {
		t.Fatalf("ListReceived: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestContactRequestRepo_ListReceived_WithStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)
	now := time.Now()

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contact_requests WHERE to_user_id = \$1 AND status = \$2`).
		WithArgs("u2", int(model.ContactRequestPending)).
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	mock.ExpectQuery(`WHERE to_user_id = \$1 AND status = \$2 ORDER BY created_at DESC LIMIT \$3 OFFSET \$4`).
		WithArgs("u2", int(model.ContactRequestPending), 10, 0).
		WillReturnRows(pgxmock.NewRows(contactRequestColumns()).
			AddRow(int64(1), "u1", "u2", int64(0), model.ContactRequestPending, "hi", now, now))

	results, total, err := repo.ListReceived(context.Background(), "u2", int(model.ContactRequestPending), 1, 10)
	if err != nil {
		t.Fatalf("ListReceived: %v", err)
	}
	if err != nil {
		t.Fatalf("ListReceived: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1", len(results))
	}
}

func TestContactRequestRepo_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectExec(`DELETE FROM contact_requests WHERE id = \$1`).
		WithArgs(int64(42)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), 42)
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestContactRequestRepo_ExistsAnyDirection(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewContactRequestRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM contacts WHERE \(user_id = \$1 AND contact_id = \$2\) OR \(user_id = \$2 AND contact_id = \$1\)`).
		WithArgs("u1", "u2").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.ExistsAnyDirection(context.Background(), "u1", "u2")
	if err != nil {
		t.Fatalf("ExistsAnyDirection: %v", err)
	}
	if !exists {
		t.Error("expected exists = true")
	}
}
