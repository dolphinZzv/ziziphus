package db

import (
	"context"
	"regexp"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
	"siciv.space/agent/panda_ai/pkg/model"
)

func TestNewReceiptRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewReceiptRepo(mock)
	if repo == nil {
		t.Fatal("NewReceiptRepo returned nil")
	}
}

func TestReceiptRepo_Upsert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewReceiptRepo(mock)
	rc := &model.Receipt{
		MsgID:     100,
		UserID:    "u1",
		SessionID: "sess_1",
		Status:    1,
		Timestamp: 5000,
	}

	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO msg_receipts (msg_id, user_id, session_id, status, timestamp) VALUES ($1, $2, $3, $4, $5) ON CONFLICT (msg_id, session_id) DO UPDATE SET status = $4, timestamp = $5`)).
		WithArgs(rc.MsgID, rc.UserID, rc.SessionID, rc.Status, rc.Timestamp).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))

	err = repo.Upsert(context.Background(), rc)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestReceiptRepo_GetByMsgID(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewReceiptRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "user_id", "session_id", "status", "timestamp"}).
		AddRow(100, "u1", "sess_1", 1, 5000).
		AddRow(100, "u2", "sess_2", 1, 5001)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT msg_id, user_id, session_id, status, timestamp FROM msg_receipts WHERE msg_id = $1`)).
		WithArgs(int64(100)).
		WillReturnRows(rows)

	receipts, err := repo.GetByMsgID(context.Background(), 100)
	if err != nil {
		t.Fatalf("GetByMsgID: %v", err)
	}
	if len(receipts) != 2 {
		t.Fatalf("got %d receipts, want 2", len(receipts))
	}
	if receipts[0].MsgID != 100 {
		t.Errorf("MsgID = %d, want 100", receipts[0].MsgID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestReceiptRepo_GetByMsgID_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewReceiptRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "user_id", "session_id", "status", "timestamp"})
	mock.ExpectQuery(`SELECT msg_id, user_id, session_id, status, timestamp FROM msg_receipts WHERE msg_id = \$1`).
		WithArgs(int64(999)).
		WillReturnRows(rows)

	receipts, err := repo.GetByMsgID(context.Background(), 999)
	if err != nil {
		t.Fatalf("GetByMsgID: %v", err)
	}
	if len(receipts) != 0 {
		t.Errorf("got %d receipts, want 0", len(receipts))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}
