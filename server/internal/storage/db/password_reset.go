package db

import (
	"context"
	"time"

	"ziziphus/pkg/model"
)

type PasswordResetRepo struct {
	pool DBPool
}

func NewPasswordResetRepo(pool DBPool) *PasswordResetRepo {
	return &PasswordResetRepo{pool: pool}
}

func (r *PasswordResetRepo) Upsert(ctx context.Context, pr *model.PasswordReset) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO password_reset (user_id, code, expires_at)
		 VALUES ($1, $2, $3)
		 ON CONFLICT (user_id) DO UPDATE SET code = $2, expires_at = $3`,
		pr.UserID, pr.Code, pr.ExpiresAt)
	return err
}

func (r *PasswordResetRepo) Get(ctx context.Context, userID string) (*model.PasswordReset, error) {
	pr := &model.PasswordReset{}
	var expiresAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT user_id, code, expires_at FROM password_reset WHERE user_id = $1`, userID,
	).Scan(&pr.UserID, &pr.Code, &expiresAt)
	if err != nil {
		return nil, err
	}
	pr.ExpiresAt = expiresAt
	return pr, nil
}

func (r *PasswordResetRepo) Delete(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM password_reset WHERE user_id = $1`, userID)
	return err
}
