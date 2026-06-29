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

func TestNewMessageRepo(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)
	if repo == nil {
		t.Fatal("NewMessageRepo returned nil")
	}
}

func TestMessageRepo_Insert(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)
	msg := &model.Message{
		MsgID:           100,
		ConvID:          "conv_1",
		SenderID:        "u1",
		SenderSessionID: "sess_1",
		ContentType:     1,
		Body:            "hello",
		Timestamp:       5000,
		ClientSeq:       10,
		ConvSeq:         5,
		Status:          1,
	}

	mock.ExpectBegin()
	mock.ExpectExec(regexp.QuoteMeta(`INSERT INTO messages (msg_id, conv_id, sender_id, sender_session_id, content_type, body, mention, reply_to, timestamp, client_seq, conv_seq, status) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`)).
		WithArgs(msg.MsgID, msg.ConvID, msg.SenderID, msg.SenderSessionID, msg.ContentType, msg.Body,
			msg.Mention, msg.ReplyTo, msg.Timestamp, msg.ClientSeq, msg.ConvSeq, msg.Status).
		WillReturnResult(pgxmock.NewResult("INSERT", 1))
	mock.ExpectExec(regexp.QuoteMeta(`UPDATE conversations SET last_msg_id = $1, last_msg_at = $2 WHERE conv_id = $3`)).
		WithArgs(msg.MsgID, time.UnixMilli(msg.Timestamp), msg.ConvID).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	err = repo.Insert(context.Background(), msg)
	if err != nil {
		t.Fatalf("Insert: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_Insert_TxError(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)
	msg := &model.Message{MsgID: 100, ConvID: "conv_1", SenderID: "u1", Timestamp: 5000}

	mock.ExpectBegin()
	mock.ExpectExec(`INSERT INTO messages`).
		WithArgs(msg.MsgID, msg.ConvID, msg.SenderID, msg.SenderSessionID, msg.ContentType, msg.Body,
			msg.Mention, msg.ReplyTo, msg.Timestamp, msg.ClientSeq, msg.ConvSeq, msg.Status).
		WillReturnError(context.DeadlineExceeded)
	mock.ExpectRollback()

	err = repo.Insert(context.Background(), msg)
	if err == nil {
		t.Fatal("expected error from Insert")
	}
}

func TestMessageRepo_GetByClientSeq(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "content_type", "body", "timestamp", "conv_seq", "status"}).
		AddRow(100, "conv_1", "u1", 1, "hello", 5000, 5, 1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT msg_id, conv_id, sender_id, content_type, body, timestamp, conv_seq, status FROM messages WHERE sender_id = $1 AND sender_session_id = $2 AND client_seq = $3`)).
		WithArgs("u1", "sess_1", int64(10)).
		WillReturnRows(rows)

	msg, err := repo.GetByClientSeq(context.Background(), "u1", "sess_1", 10)
	if err != nil {
		t.Fatalf("GetByClientSeq: %v", err)
	}
	if msg.MsgID != 100 {
		t.Errorf("MsgID = %d, want 100", msg.MsgID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetByClientSeq_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectQuery(`SELECT msg_id, conv_id, sender_id, content_type, body, timestamp, conv_seq, status FROM messages WHERE sender_id = \$1 AND sender_session_id = \$2 AND client_seq = \$3`).
		WithArgs("u1", "sess_1", int64(99)).
		WillReturnError(pgx.ErrNoRows)

	msg, err := repo.GetByClientSeq(context.Background(), "u1", "sess_1", 99)
	if err == nil {
		t.Fatal("expected error for not found")
	}
	if msg != nil {
		t.Error("expected nil message")
	}
}

func TestMessageRepo_GetHistory(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(102, "conv_1", "u1", "u1", 1, "msg2", nil, nil, 5002, 2, 1).
		AddRow(101, "conv_1", "u2", "u2", 1, "msg1", nil, nil, 5001, 1, 1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status FROM messages m LEFT JOIN users u ON u.id = m.sender_id WHERE m.conv_id = $1 AND m.deleted = false AND m.msg_id < $2 ORDER BY m.msg_id DESC LIMIT $3`)).
		WithArgs("conv_1", int64(200), 20).
		WillReturnRows(rows)

	msgs, err := repo.GetHistory(context.Background(), "conv_1", 200, 0, 20, "", 0, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetHistory_NoBefore(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(101, "conv_1", "u1", "u1", 1, "msg1", nil, nil, 5001, 1, 1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status FROM messages m LEFT JOIN users u ON u.id = m.sender_id WHERE m.conv_id = $1 AND m.deleted = false ORDER BY m.msg_id DESC LIMIT $2`)).
		WithArgs("conv_1", 20).
		WillReturnRows(rows)

	msgs, err := repo.GetHistory(context.Background(), "conv_1", 0, 0, 20, "", 0, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("got %d messages, want 1", len(msgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetHistory_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"})
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status FROM messages m LEFT JOIN users u ON u.id = m.sender_id WHERE m.conv_id = $1 AND m.deleted = false ORDER BY m.msg_id DESC LIMIT $2`)).
		WithArgs("conv_empty", 20).
		WillReturnRows(rows)

	msgs, err := repo.GetHistory(context.Background(), "conv_empty", 0, 0, 20, "", 0, 0)
	if err != nil {
		t.Fatalf("GetHistory: %v", err)
	}
	if len(msgs) != 0 {
		t.Errorf("got %d messages, want 0", len(msgs))
	}
}

func TestMessageRepo_GetMessagesSinceSeq(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(102, "conv_1", "u1", "user1", 1, "msg2", nil, nil, 5002, 2, 1).
		AddRow(103, "conv_1", "u2", "user2", 1, "msg3", nil, nil, 5003, 3, 1)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status FROM messages m LEFT JOIN users u ON u.id = m.sender_id WHERE m.conv_id = $1 AND m.conv_seq > $2 AND m.deleted = false ORDER BY m.conv_seq ASC LIMIT $3`)).
		WithArgs("conv_1", int64(1), 50).
		WillReturnRows(rows)

	msgs, err := repo.GetMessagesSinceSeq(context.Background(), "conv_1", 1, 50)
	if err != nil {
		t.Fatalf("GetMessagesSinceSeq: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetMaxConvSeq(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT COALESCE(MAX(conv_seq), 0) FROM messages WHERE conv_id = $1`)).
		WithArgs("conv_1").
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow(42))

	seq, err := repo.GetMaxConvSeq(context.Background(), "conv_1")
	if err != nil {
		t.Fatalf("GetMaxConvSeq: %v", err)
	}
	if seq != 42 {
		t.Errorf("seq = %d, want 42", seq)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetMaxConvSeq_Empty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectQuery(`SELECT COALESCE\(MAX\(conv_seq\), 0\) FROM messages WHERE conv_id = \$1`).
		WithArgs("conv_empty").
		WillReturnRows(pgxmock.NewRows([]string{"max"}).AddRow(0))

	seq, err := repo.GetMaxConvSeq(context.Background(), "conv_empty")
	if err != nil {
		t.Fatalf("GetMaxConvSeq: %v", err)
	}
	if seq != 0 {
		t.Errorf("seq = %d, want 0", seq)
	}
}

func TestMessageRepo_Get(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	rows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status", "deleted"}).
		AddRow(100, "conv_1", "u1", 1, "hello", nil, nil, 5000, 5, 1, false)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status, deleted FROM messages WHERE msg_id = $1`)).
		WithArgs(int64(100)).
		WillReturnRows(rows)

	msg, err := repo.Get(context.Background(), 100)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if msg.MsgID != 100 {
		t.Errorf("MsgID = %d, want 100", msg.MsgID)
	}
	if msg.Deleted {
		t.Error("deleted should be false")
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetHistory_Around(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)
	half := 2 // limit=4 -> half=2

	// "before" query: messages before aroundMsgID, DESC, limit=half
	beforeRows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(102, "conv_1", "u2", "u2", 1, "msg2", nil, nil, 5002, 2, 1).
		AddRow(101, "conv_1", "u1", "u1", 1, "msg1", nil, nil, 5001, 1, 1)

	mock.ExpectQuery(`AND m\.msg_id < \$2 ORDER BY m\.msg_id DESC LIMIT \$3`).
		WithArgs("conv_1", int64(200), half).
		WillReturnRows(beforeRows)

	// "after" query: messages from aroundMsgID onward, ASC, limit=half
	afterRows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(200, "conv_1", "u1", "u1", 1, "around", nil, nil, 6000, 3, 1).
		AddRow(201, "conv_1", "u2", "u2", 1, "next", nil, nil, 6001, 4, 1)

	mock.ExpectQuery(`AND m\.msg_id >= \$2 ORDER BY m\.msg_id ASC LIMIT \$3`).
		WithArgs("conv_1", int64(200), half).
		WillReturnRows(afterRows)

	msgs, err := repo.GetHistory(context.Background(), "conv_1", 0, 200, 4, "", 0, 0)
	if err != nil {
		t.Fatalf("GetHistory around: %v", err)
	}
	// Expected: before reversed [101, 102] + after [200, 201] = [101, 102, 200, 201]
	if len(msgs) != 4 {
		t.Fatalf("got %d messages, want 4", len(msgs))
	}
	if msgs[0].MsgID != 101 {
		t.Errorf("msgs[0].MsgID = %d, want 101", msgs[0].MsgID)
	}
	if msgs[1].MsgID != 102 {
		t.Errorf("msgs[1].MsgID = %d, want 102", msgs[1].MsgID)
	}
	if msgs[2].MsgID != 200 {
		t.Errorf("msgs[2].MsgID = %d, want 200", msgs[2].MsgID)
	}
	if msgs[3].MsgID != 201 {
		t.Errorf("msgs[3].MsgID = %d, want 201", msgs[3].MsgID)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_GetHistory_Around_WithFilters(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)
	half := 1 // limit=2 -> half=1

	// args order: $1=convID, $2=keyword, $3=startDate, $4=endDate, $5=aroundMsgID, $6=half
	beforeRows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(101, "conv_1", "u1", "u1", 1, "hello", nil, nil, 5001, 1, 1)

	mock.ExpectQuery(`AND m\.msg_id < \$5 ORDER BY m\.msg_id DESC LIMIT \$6`).
		WithArgs("conv_1", "%hello%", int64(5000), int64(6000), int64(200), half).
		WillReturnRows(beforeRows)

	afterRows := pgxmock.NewRows([]string{"msg_id", "conv_id", "sender_id", "sender_name", "content_type", "body", "mention", "reply_to", "timestamp", "conv_seq", "status"}).
		AddRow(200, "conv_1", "u1", "u1", 1, "hello", nil, nil, 6000, 3, 1)

	mock.ExpectQuery(`AND m\.msg_id >= \$5 ORDER BY m\.msg_id ASC LIMIT \$6`).
		WithArgs("conv_1", "%hello%", int64(5000), int64(6000), int64(200), half).
		WillReturnRows(afterRows)

	msgs, err := repo.GetHistory(context.Background(), "conv_1", 0, 200, 2, "hello", 5000, 6000)
	if err != nil {
		t.Fatalf("GetHistory around with filters: %v", err)
	}
	if len(msgs) != 2 {
		t.Fatalf("got %d messages, want 2", len(msgs))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_UpdateBody(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE messages SET body = $1, content_type = $2 WHERE msg_id = $3`)).
		WithArgs("edited text", model.ContentEdit, int64(100)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.UpdateBody(context.Background(), 100, "edited text")
	if err != nil {
		t.Fatalf("UpdateBody: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_Recall(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE messages SET content_type = $1, body = '' WHERE msg_id = $2`)).
		WithArgs(model.ContentRecall, int64(200)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	err = repo.Recall(context.Background(), 200)
	if err != nil {
		t.Fatalf("Recall: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("expectations not met: %v", err)
	}
}

func TestMessageRepo_Get_NotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatalf("pgxmock.NewPool: %v", err)
	}
	defer mock.Close()

	repo := NewMessageRepo(mock)

	mock.ExpectQuery(`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status, deleted FROM messages WHERE msg_id = \$1`).
		WithArgs(int64(999)).
		WillReturnError(pgx.ErrNoRows)

	_, err = repo.Get(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for not found")
	}
}
