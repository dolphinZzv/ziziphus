package db

import (
	"context"

	"siciv.space/agent/panda_ai/pkg/model"
)

type ReceiptRepo struct {
	pool DBPool
}

func NewReceiptRepo(pool DBPool) *ReceiptRepo {
	return &ReceiptRepo{pool: pool}
}

func (r *ReceiptRepo) Upsert(ctx context.Context, receipt *model.Receipt) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO msg_receipts (msg_id, user_id, session_id, status, timestamp)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (msg_id, session_id) DO UPDATE SET status = $4, timestamp = $5`,
		receipt.MsgID, receipt.UserID, receipt.SessionID, receipt.Status, receipt.Timestamp)
	return err
}

func (r *ReceiptRepo) GetByMsgID(ctx context.Context, msgID int64) ([]*model.Receipt, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT msg_id, user_id, session_id, status, timestamp FROM msg_receipts WHERE msg_id = $1`, msgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var receipts []*model.Receipt
	for rows.Next() {
		rc := &model.Receipt{}
		if err := rows.Scan(&rc.MsgID, &rc.UserID, &rc.SessionID, &rc.Status, &rc.Timestamp); err != nil {
			return nil, err
		}
		receipts = append(receipts, rc)
	}
	return receipts, nil
}
