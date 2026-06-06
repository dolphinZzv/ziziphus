package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/dolphinz/im-server/pkg/model"
)

type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, type, name, avatar, status, password, ext_meta, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		u.ID, u.Type, u.Name, u.Avatar, u.Status, u.Password, u.ExtMeta, time.UnixMilli(u.CreatedAt))
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, name, avatar, status, password, ext_meta, created_at FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &u.Password, &u.ExtMeta, &createdAt)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

func (r *UserRepo) GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, avatar, status, created_at FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*model.User, len(ids))
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		result[u.ID] = u
	}
	return result, nil
}

func (r *UserRepo) Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error) {
	count := 0
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE name ILIKE $1`, "%"+q+"%").Scan(&count)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, avatar, status, created_at FROM users WHERE name ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		"%"+q+"%", size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt); err != nil {
			return nil, 0, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		users = append(users, u)
	}
	return users, count, nil
}

func (r *UserRepo) Update(ctx context.Context, id, name, avatar string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $1, avatar = $2 WHERE id = $3`, name, avatar, id)
	return err
}
