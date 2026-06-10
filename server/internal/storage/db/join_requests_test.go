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

func TestNewJoinRequestRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)
	if repo == nil {
		t.Fatal("NewJoinRequestRepo returned nil")
	}
}

func TestJoinRequestRepo_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO join_requests (conv_id, user_id) VALUES ($1, $2) ON CONFLICT (conv_id, user_id) DO NOTHING`)).
		WithArgs("conv_1", "u1").
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Create(context.Background(), "conv_1", "u1")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"conv_id", "user_id", "status", "created_at", "updated_at"}).
		AddRow("conv_1", "u1", 0, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT conv_id, user_id, status, created_at, updated_at FROM join_requests WHERE conv_id = $1 AND user_id = $2`)).
		WithArgs("conv_1", "u1").
		WillReturnRows(rows)

	jr, err := repo.Get(context.Background(), "conv_1", "u1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if jr == nil {
		t.Fatal("expected non-nil join request")
	}
	if jr.ConvID != "conv_1" {
		t.Errorf("ConvID = %q, want conv_1", jr.ConvID)
	}
	if jr.UserID != "u1" {
		t.Errorf("UserID = %q, want u1", jr.UserID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_Get_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectQuery(`SELECT conv_id, user_id, status, created_at, updated_at FROM join_requests WHERE conv_id = \$1 AND user_id = \$2`).
		WithArgs("conv_x", "u_x").
		WillReturnError(pgx.ErrNoRows)

	jr, err := repo.Get(context.Background(), "conv_x", "u_x")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if jr != nil {
		t.Error("expected nil when no rows")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_Get_DBError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectQuery(`SELECT conv_id, user_id, status, created_at, updated_at FROM join_requests WHERE conv_id = \$1 AND user_id = \$2`).
		WithArgs("conv_1", "u1").
		WillReturnError(context.DeadlineExceeded)

	_, err = repo.Get(context.Background(), "conv_1", "u1")
	if err == nil {
		t.Fatal("expected error from DB")
	}
}

func TestJoinRequestRepo_ListByConv(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)
	now := time.Now()

	rows := pgxmock.NewRows([]string{"conv_id", "user_id", "status", "created_at", "updated_at"}).
		AddRow("conv_1", "u1", 0, now, now).
		AddRow("conv_1", "u2", 0, now, now)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT conv_id, user_id, status, created_at, updated_at FROM join_requests WHERE conv_id = $1 AND status = $2 ORDER BY created_at`)).
		WithArgs("conv_1", model.JoinRequestStatus(0)).
		WillReturnRows(rows)

	requests, err := repo.ListByConv(context.Background(), "conv_1", 0)
	if err != nil {
		t.Fatalf("ListByConv: %v", err)
	}
	if len(requests) != 2 {
		t.Fatalf("got %d requests, want 2", len(requests))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_ListByConv_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	rows := pgxmock.NewRows([]string{"conv_id", "user_id", "status", "created_at", "updated_at"})
	mock.ExpectQuery(`SELECT conv_id, user_id, status, created_at, updated_at FROM join_requests WHERE conv_id = \$1 AND status = \$2 ORDER BY created_at`).
		WithArgs("conv_empty", model.JoinRequestStatus(0)).
		WillReturnRows(rows)

	requests, err := repo.ListByConv(context.Background(), "conv_empty", 0)
	if err != nil {
		t.Fatalf("ListByConv: %v", err)
	}
	if len(requests) != 0 {
		t.Errorf("got %d requests, want 0", len(requests))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_UpdateStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE join_requests SET status = $3, updated_at = NOW() WHERE conv_id = $1 AND user_id = $2`)).
		WithArgs("conv_1", "u1", model.JoinRequestStatus(1)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateStatus(context.Background(), "conv_1", "u1", 1)
	if err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_ExistsPending(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT(*) FROM join_requests WHERE conv_id = $1 AND user_id = $2 AND status = 0`)).
		WithArgs("conv_1", "u1").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(1))

	exists, err := repo.ExistsPending(context.Background(), "conv_1", "u1")
	if err != nil {
		t.Fatalf("ExistsPending: %v", err)
	}
	if !exists {
		t.Error("expected pending request to exist")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestJoinRequestRepo_ExistsPending_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewJoinRequestRepo(mock)

	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM join_requests WHERE conv_id = \$1 AND user_id = \$2 AND status = 0`).
		WithArgs("conv_1", "u_x").
		WillReturnRows(pgxmock.NewRows([]string{"count"}).AddRow(0))

	exists, err := repo.ExistsPending(context.Background(), "conv_1", "u_x")
	if err != nil {
		t.Fatalf("ExistsPending: %v", err)
	}
	if exists {
		t.Error("expected no pending request")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
