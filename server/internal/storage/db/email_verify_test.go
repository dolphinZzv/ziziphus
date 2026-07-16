package db

import (
	"context"
	"regexp"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
	"ziziphus/pkg/model"
)

func TestNewEmailVerifyRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewEmailVerifyRepo(mock)
	if repo == nil {
		t.Fatal("NewEmailVerifyRepo returned nil")
	}
}

func TestEmailVerifyRepo_Upsert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewEmailVerifyRepo(mock)
	now := time.Now()

	ev := &model.EmailVerify{
		UserID:       "u1",
		PendingEmail: "new@example.com",
		Code:         "123456",
		ExpiresAt:    now,
	}

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO email_verify (user_id, pending_email, code, expires_at) VALUES ($1, $2, $3, $4) ON CONFLICT (user_id) DO UPDATE SET pending_email = $2, code = $3, expires_at = $4`,
	)).
		WithArgs(ev.UserID, ev.PendingEmail, ev.Code, ev.ExpiresAt).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Upsert(context.Background(), ev)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestEmailVerifyRepo_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewEmailVerifyRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"user_id", "pending_email", "code", "expires_at"}).
		AddRow("u1", "new@example.com", "123456", now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, pending_email, code, expires_at FROM email_verify WHERE user_id = $1`,
	)).
		WithArgs("u1").
		WillReturnRows(rows)

	ev, err := repo.Get(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ev == nil {
		t.Fatal("Get returned nil")
	}
	if ev.UserID != "u1" {
		t.Errorf("UserID = %q, want u1", ev.UserID)
	}
	if ev.PendingEmail != "new@example.com" {
		t.Errorf("PendingEmail = %q, want new@example.com", ev.PendingEmail)
	}
	if ev.Code != "123456" {
		t.Errorf("Code = %q, want 123456", ev.Code)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestEmailVerifyRepo_Get_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewEmailVerifyRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, pending_email, code, expires_at FROM email_verify WHERE user_id = $1`,
	)).
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	ev, err := repo.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Get expected error, got nil")
	}
	if ev != nil {
		t.Errorf("Get expected nil result, got %v", ev)
	}
}

func TestEmailVerifyRepo_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewEmailVerifyRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM email_verify WHERE user_id = $1`)).
		WithArgs("u1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), "u1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
