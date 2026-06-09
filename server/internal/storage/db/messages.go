package db

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type MessageRepo struct {
	pool *pgxpool.Pool
}

func NewMessageRepo(pool *pgxpool.Pool) *MessageRepo {
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
		`UPDATE conversations SET last_msg_id = $1, last_msg_at = to_timestamp($2::numeric / 1000.0) WHERE conv_id = $3`,
		msg.MsgID, msg.Timestamp, msg.ConvID)
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

func (r *MessageRepo) GetHistory(ctx context.Context, convID string, beforeMsgID int64, limit int) ([]*model.Message, error) {
	var rows pgx.Rows
	var err error
	if beforeMsgID > 0 {
		rows, err = r.pool.Query(ctx,
			`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status
			 FROM messages WHERE conv_id = $1 AND msg_id < $2 AND deleted = false
			 ORDER BY msg_id DESC LIMIT $3`, convID, beforeMsgID, limit)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT msg_id, conv_id, sender_id, content_type, body, mention, reply_to, timestamp, conv_seq, status
			 FROM messages WHERE conv_id = $1 AND deleted = false
			 ORDER BY msg_id DESC LIMIT $2`, convID, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanMessages(rows)
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
