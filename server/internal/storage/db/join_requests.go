package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type JoinRequestRepo struct {
	pool *pgxpool.Pool
}

func NewJoinRequestRepo(pool *pgxpool.Pool) *JoinRequestRepo {
	return &JoinRequestRepo{pool: pool}
}

func (r *JoinRequestRepo) Create(ctx context.Context, convID, userID string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO join_requests (conv_id, user_id)
		 VALUES ($1, $2)
		 ON CONFLICT (conv_id, user_id) DO NOTHING`,
		convID, userID)
	return err
}

func (r *JoinRequestRepo) Get(ctx context.Context, convID, userID string) (*model.JoinRequest, error) {
	jr := &model.JoinRequest{}
	var createdAt, updatedAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT conv_id, user_id, status, created_at, updated_at
		 FROM join_requests WHERE conv_id = $1 AND user_id = $2`,
		convID, userID).
		Scan(&jr.ConvID, &jr.UserID, &jr.Status, &createdAt, &updatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	jr.CreatedAt = createdAt.UnixMilli()
	jr.UpdatedAt = updatedAt.UnixMilli()
	return jr, nil
}

func (r *JoinRequestRepo) ListByConv(ctx context.Context, convID string, status model.JoinRequestStatus) ([]*model.JoinRequest, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT conv_id, user_id, status, created_at, updated_at
		 FROM join_requests WHERE conv_id = $1 AND status = $2
		 ORDER BY created_at`, convID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*model.JoinRequest
	for rows.Next() {
		jr := &model.JoinRequest{}
		var createdAt, updatedAt time.Time
		if err := rows.Scan(&jr.ConvID, &jr.UserID, &jr.Status, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		jr.CreatedAt = createdAt.UnixMilli()
		jr.UpdatedAt = updatedAt.UnixMilli()
		result = append(result, jr)
	}
	return result, nil
}

func (r *JoinRequestRepo) UpdateStatus(ctx context.Context, convID, userID string, status model.JoinRequestStatus) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE join_requests SET status = $3, updated_at = NOW()
		 WHERE conv_id = $1 AND user_id = $2`,
		convID, userID, status)
	return err
}

func (r *JoinRequestRepo) ExistsPending(ctx context.Context, convID, userID string) (bool, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM join_requests
		 WHERE conv_id = $1 AND user_id = $2 AND status = 0`,
		convID, userID).Scan(&count)
	return count > 0, err
}
