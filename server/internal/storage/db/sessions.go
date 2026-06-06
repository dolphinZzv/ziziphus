package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type SessionRepo struct {
	pool *pgxpool.Pool
}

func NewSessionRepo(pool *pgxpool.Pool) *SessionRepo {
	return &SessionRepo{pool: pool}
}

func (r *SessionRepo) Create(ctx context.Context, s *model.Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO sessions (session_id, user_id, device, device_name, status, login_at, last_active)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		s.SessionID, s.UserID, s.Device, s.DeviceName, s.Status,
		time.UnixMilli(s.LoginAt), time.UnixMilli(s.LastActive))
	return err
}

func (r *SessionRepo) Delete(ctx context.Context, sessionID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE session_id = $1`, sessionID)
	return err
}
