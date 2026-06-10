package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/model"
)

type MessageRepo struct {
	pool DBPool
}

func NewMessageRepo(pool DBPool) *MessageRepo {
	return &MessageRepo{pool: pool}
}

func (r *MessageRepo) Insert(ctx context.Context, msg *model.Message) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`INSERT INTO messages (msg_id, conv_id, sender_id, sender_session_id, content_type, body, mention, reply_to, timestamp, client_seq, conv_seq, status)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		msg.MsgID, msg.ConvID, msg.SenderID, msg.SenderSessionID, msg.ContentType, msg.Body,
		msg.Mention, msg.ReplyTo, msg.Timestamp, msg.ClientSeq, msg.ConvSeq, msg.Status)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx,
		`UPDATE conversations SET last_msg_id = $1, last_msg_at = $2 WHERE conv_id = $3`,
		msg.MsgID, time.UnixMilli(msg.Timestamp), msg.ConvID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *MessageRepo) GetByClientSeq(ctx context.Context, senderID, sessionID string, clientSeq int64) (*model.Message, error) {
	msg := &model.Message{}
	err := r.pool.QueryRow(ctx,
		`SELECT msg_id, conv_id, sender_id, content_type, body, timestamp, conv_seq, status
		 FROM messages WHERE sender_id = $1 AND sender_session_id = $2 AND client_seq = $3`,
		senderID, sessionID, clientSeq).
		Scan(&msg.MsgID, &msg.ConvID, &msg.SenderID, &msg.ContentType, &msg.Body, &msg.Timestamp, &msg.ConvSeq, &msg.Status)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (r *MessageRepo) GetHistory(ctx context.Context, convID string, beforeMsgID, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
	if aroundMsgID > 0 {
		return r.getHistoryAround(ctx, convID, aroundMsgID, limit, keyword, startDate, endDate)
	}

	query := `SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status
		 FROM messages m
		 LEFT JOIN users u ON u.id = m.sender_id
		 WHERE m.conv_id = $1 AND m.deleted = false`
	args := []interface{}{convID}
	argIdx := 2

	if beforeMsgID > 0 {
		query += fmt.Sprintf(" AND m.msg_id < $%d", argIdx)
		args = append(args, beforeMsgID)
		argIdx++
	}
	if keyword != "" {
		query += fmt.Sprintf(" AND m.body ILIKE $%d", argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	if startDate > 0 {
		query += fmt.Sprintf(" AND m.timestamp >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate > 0 {
		query += fmt.Sprintf(" AND m.timestamp <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY m.msg_id DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanHistoryMessages(rows)
}

func (r *MessageRepo) getHistoryAround(ctx context.Context, convID string, aroundMsgID int64, limit int, keyword string, startDate, endDate int64) ([]*model.Message, error) {
	q := `SELECT m.msg_id, m.conv_id, m.sender_id, COALESCE(u.name, ''), m.content_type, m.body, m.mention, m.reply_to, m.timestamp, m.conv_seq, m.status
		 FROM messages m
		 LEFT JOIN users u ON u.id = m.sender_id
		 WHERE m.conv_id = $1 AND m.deleted = false`
	args := []interface{}{convID}
	argIdx := 2

	if keyword != "" {
		q += fmt.Sprintf(" AND m.body ILIKE $%d", argIdx)
		args = append(args, "%"+keyword+"%")
		argIdx++
	}
	if startDate > 0 {
		q += fmt.Sprintf(" AND m.timestamp >= $%d", argIdx)
		args = append(args, startDate)
		argIdx++
	}
	if endDate > 0 {
		q += fmt.Sprintf(" AND m.timestamp <= $%d", argIdx)
		args = append(args, endDate)
		argIdx++
	}

	half := limit / 2
	if half < 1 {
		half = 1
	}

	// Messages before the target (ordered DESC, take half)
	beforeQ := q + fmt.Sprintf(" AND m.msg_id < $%d ORDER BY m.msg_id DESC LIMIT $%d", argIdx, argIdx+1)
	beforeArgs := append(append([]interface{}{}, args...), aroundMsgID, half)
	beforeRows, err := r.pool.Query(ctx, beforeQ, beforeArgs...)
	if err != nil {
		return nil, err
	}
	defer beforeRows.Close()
	before, err := scanHistoryMessages(beforeRows)
	if err != nil {
		return nil, err
	}

	// Messages from the target onward (ordered ASC, take half)
	afterQ := q + fmt.Sprintf(" AND m.msg_id >= $%d ORDER BY m.msg_id ASC LIMIT $%d", argIdx, argIdx+1)
	afterArgs := append(append([]interface{}{}, args...), aroundMsgID, half)
	afterRows, err := r.pool.Query(ctx, afterQ, afterArgs...)
	if err != nil {
		return nil, err
	}
	defer afterRows.Close()
	after, err := scanHistoryMessages(afterRows)
	if err != nil {
		return nil, err
	}

	// before is DESC, so reverse it, then append after
	merged := make([]*model.Message, 0, len(before)+len(after))
	for i := len(before) - 1; i >= 0; i-- {
		merged = append(merged, before[i])
	}
	merged = append(merged, after...)
	return merged, nil
}

func (r *MessageRepo) GetMessagesSinceSeq(ctx context.Context, convID string, lastSeq int64, limit int) ([]*model.Message, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status
		 FROM messages WHERE conv_id = $1 AND conv_seq > $2 AND deleted = false
		 ORDER BY conv_seq ASC LIMIT $3`, convID, lastSeq, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
}

func (r *MessageRepo) GetMaxConvSeq(ctx context.Context, convID string) (int64, error) {
	var seq int64
	err := r.pool.QueryRow(ctx,
		`SELECT COALESCE(MAX(conv_seq), 0) FROM messages WHERE conv_id = $1`, convID).Scan(&seq)
	return seq, err
}

func (r *MessageRepo) Get(ctx context.Context, msgID int64) (*model.Message, error) {
	msg := &model.Message{}
	err := r.pool.QueryRow(ctx,
		`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status, deleted
		 FROM messages WHERE msg_id = $1`, msgID).
		Scan(&msg.MsgID, &msg.ConvID, &msg.SenderID, &msg.ContentType, &msg.Body,
			&msg.Mention, &msg.ReplyTo, &msg.Timestamp, &msg.ConvSeq, &msg.Status, &msg.Deleted)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func scanMessages(rows pgx.Rows) ([]*model.Message, error) {
	var msgs []*model.Message
	for rows.Next() {
		msg := &model.Message{}
		if err := rows.Scan(&msg.MsgID, &msg.ConvID, &msg.SenderID, &msg.ContentType,
			&msg.Body, &msg.Mention, &msg.ReplyTo, &msg.Timestamp, &msg.ConvSeq, &msg.Status); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func scanHistoryMessages(rows pgx.Rows) ([]*model.Message, error) {
	var msgs []*model.Message
	for rows.Next() {
		msg := &model.Message{}
		if err := rows.Scan(&msg.MsgID, &msg.ConvID, &msg.SenderID, &msg.SenderName,
			&msg.ContentType, &msg.Body, &msg.Mention, &msg.ReplyTo, &msg.Timestamp, &msg.ConvSeq, &msg.Status); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}
