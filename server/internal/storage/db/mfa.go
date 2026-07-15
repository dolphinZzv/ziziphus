package db

import (
	"context"
	"time"

	"ziziphus/pkg/model"
)

type MFARepo struct {
	pool DBPool
}

func NewMFARepo(pool DBPool) *MFARepo {
	return &MFARepo{pool: pool}
}

func (r *MFARepo) Get(ctx context.Context, userID string) (*model.UserMFA, error) {
	m := &model.UserMFA{}
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, mfa_type, enabled, secret, created_at, updated_at FROM user_mfa WHERE user_id = $1`, userID,
	).Scan(&m.UserID, &m.MFAType, &m.Enabled, &m.Secret, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}
	return m, nil
}

func (r *MFARepo) Upsert(ctx context.Context, m *model.UserMFA) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO user_mfa (user_id, mfa_type, enabled, secret, updated_at)
		 VALUES ($1, $2, $3, $4, NOW())
		 ON CONFLICT (user_id) DO UPDATE SET mfa_type = $2, enabled = $3, secret = $4, updated_at = NOW()`,
		m.UserID, m.MFAType, m.Enabled, m.Secret)
	return err
}

func (r *MFARepo) Disable(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE user_mfa SET enabled = FALSE, updated_at = NOW() WHERE user_id = $1`, userID)
	return err
}
