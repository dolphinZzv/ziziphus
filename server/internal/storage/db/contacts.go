package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type ContactRepo struct {
	pool *pgxpool.Pool
}

func NewContactRepo(pool *pgxpool.Pool) *ContactRepo {
	return &ContactRepo{pool: pool}
}

func (r *ContactRepo) Add(ctx context.Context, c *model.Contact) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO contacts (user_id, contact_id, nickname, added_at)
		 VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, contact_id) DO NOTHING`,
		c.UserID, c.ContactID, c.Nickname, time.UnixMilli(c.AddedAt))
	return err
}

func (r *ContactRepo) Remove(ctx context.Context, userID, contactID string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM contacts WHERE user_id = $1 AND contact_id = $2`, userID, contactID)
	return err
}

func (r *ContactRepo) List(ctx context.Context, userID string, page, size int) ([]*model.Contact, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM contacts WHERE user_id = $1`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT user_id, contact_id, nickname, added_at FROM contacts WHERE user_id = $1
		 ORDER BY added_at DESC LIMIT $2 OFFSET $3`, userID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var contacts []*model.Contact
	for rows.Next() {
		c := &model.Contact{}
		var addedAt time.Time
		if err := rows.Scan(&c.UserID, &c.ContactID, &c.Nickname, &addedAt); err != nil {
			return nil, 0, err
		}
		c.AddedAt = addedAt.UnixMilli()
		contacts = append(contacts, c)
	}
	return contacts, total, nil
}

func (r *ContactRepo) UpdateNickname(ctx context.Context, userID, contactID, nickname string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE contacts SET nickname = $1 WHERE user_id = $2 AND contact_id = $3`,
		nickname, userID, contactID)
	return err
}
