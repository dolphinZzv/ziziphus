package db

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewSessionRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewSessionRepo(mock)
	if repo == nil {
		t.Fatal("NewSessionRepo returned nil")
	}
}

func TestSessionRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewSessionRepo(mock)
	s := &model.Session{
		SessionID:  "sess_1",
		UserID:     "u1",
		Device:     1,
		DeviceName: "ios",
		Status:     1,
		LoginAt:    1000,
		LastActive: 2000,
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO sessions (session_id, user_id, device, device_name, status, login_at, last_active) VALUES ($1, $2, $3, $4, $5, $6, $7)`)).
		WithArgs(s.SessionID, s.UserID, s.Device, s.DeviceName, s.Status, AnyTime{}, AnyTime{}).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), s)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestSessionRepo_Delete(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewSessionRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`DELETE FROM sessions WHERE session_id = $1`)).
		WithArgs("sess_1").
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	err = repo.Delete(context.Background(), "sess_1")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
