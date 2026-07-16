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

func TestMFARepo_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"user_id", "mfa_type", "enabled", "secret", "created_at", "updated_at"}).
		AddRow("user_1", model.MFATOTP, true, "testsecret", now, now)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, mfa_type, enabled, secret, created_at, updated_at FROM user_mfa WHERE user_id = $1`,
	)).
		WithArgs("user_1").
		WillReturnRows(rows)

	m, err := repo.Get(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if m == nil {
		t.Fatal("Get returned nil")
	}
	if m.UserID != "user_1" {
		t.Errorf("UserID = %q, want user_1", m.UserID)
	}
	if m.MFAType != model.MFATOTP {
		t.Errorf("MFAType = %d, want %d", m.MFAType, model.MFATOTP)
	}
	if !m.Enabled {
		t.Error("Enabled = false, want true")
	}
	if m.Secret != "testsecret" {
		t.Errorf("Secret = %q, want testsecret", m.Secret)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMFARepo_Get_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(
		`SELECT user_id, mfa_type, enabled, secret, created_at, updated_at FROM user_mfa WHERE user_id = $1`,
	)).
		WithArgs("nonexistent").
		WillReturnError(pgx.ErrNoRows)

	m, err := repo.Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Get expected error for non-existent user, got nil")
	}
	if m != nil {
		t.Errorf("Get expected nil, got %v", m)
	}
}

func TestMFARepo_Upsert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(
		`INSERT INTO user_mfa (user_id, mfa_type, enabled, secret, updated_at) VALUES ($1, $2, $3, $4, NOW()) ON CONFLICT (user_id) DO UPDATE SET mfa_type = $2, enabled = $3, secret = $4, updated_at = NOW()`)).
		WithArgs("user_1", model.MFATOTP, true, "testsecret").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	m := &model.UserMFA{
		UserID:  "user_1",
		MFAType: model.MFATOTP,
		Enabled: true,
		Secret:  "testsecret",
	}
	err = repo.Upsert(context.Background(), m)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMFARepo_Disable(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMFARepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE user_mfa SET enabled = FALSE, updated_at = NOW() WHERE user_id = $1`)).
		WithArgs("user_1").
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Disable(context.Background(), "user_1")
	if err != nil {
		t.Fatalf("Disable: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
