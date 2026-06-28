package db

import (
	"context"
	"time"

	"siciv.space/agent/panda_ai/pkg/model"
)

type EmailVerifyRepo struct {
	pool DBPool
}

func NewEmailVerifyRepo(pool DBPool) *EmailVerifyRepo {
	return &EmailVerifyRepo{pool: pool}
}

func (r *EmailVerifyRepo) Upsert(ctx context.Context, ev *model.EmailVerify) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO email_verify (user_id, pending_email, code, expires_at)
		 VALUES ($1, $2, $3, $4)
		 ON CONFLICT (user_id) DO UPDATE SET pending_email = $2, code = $3, expires_at = $4`,
		ev.UserID, ev.PendingEmail, ev.Code, ev.ExpiresAt)
	return err
}

func (r *EmailVerifyRepo) Get(ctx context.Context, userID string) (*model.EmailVerify, error) {
	ev := &model.EmailVerify{}
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, pending_email, code, expires_at FROM email_verify WHERE user_id = $1`, userID,
	).Scan(&ev.UserID, &ev.PendingEmail, &ev.Code, &expiresAt)
	if err != nil {
		return nil, err
	}
	ev.ExpiresAt = expiresAt
	return ev, nil
}

func (r *EmailVerifyRepo) Delete(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM email_verify WHERE user_id = $1`, userID)
	return err
}
