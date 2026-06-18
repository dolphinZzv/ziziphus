package db

import (
	"context"
	"time"

	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

type UserRepo struct {
	pool DBPool
}

func NewUserRepo(pool DBPool) *UserRepo {
	return &UserRepo{pool: pool}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		u.ID, u.Type, u.Name, u.Avatar, u.Status, u.Password, u.ExtMeta, time.UnixMilli(u.CreatedAt), u.Account, u.PrimaryColor, u.SecondaryColor, u.UID, u.WakeMode, u.APIKey)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &u.Password, &u.ExtMeta, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey)
	if err != nil {
		logger.Error("GetByID query failed",
			"id", id,
			"error", err)
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

func (r *UserRepo) GetByIDs(ctx context.Context, ids []string) (map[string]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*model.User, len(ids))
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey); err != nil {
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
		`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE name ILIKE $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		"%"+q+"%", size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey); err != nil {
			return nil, 0, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		users = append(users, u)
	}
	return users, count, nil
}

func (r *UserRepo) GetByAccount(ctx context.Context, account string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, name, avatar, status, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE account = $1`, account).
		Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &u.Password, &u.ExtMeta, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, id, name, avatar, primaryColor, secondaryColor string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $1, avatar = $2, primary_color = $3, secondary_color = $4 WHERE id = $5`,
		name, avatar, primaryColor, secondaryColor, id)
	return err
}

// CountAgents returns the number of agents owned by uid.
func (r *UserRepo) CountAgents(ctx context.Context, uid string) (int, error) {
	count := 0
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM users WHERE type = $1 AND uid = $2`, model.UserAgent, uid).Scan(&count)
	return count, err
}

// ListAgents lists agents owned by uid.
func (r *UserRepo) ListAgents(ctx context.Context, uid string) ([]*model.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE type = $1 AND uid = $2 ORDER BY created_at ASC`,
		model.UserAgent, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		users = append(users, u)
	}
	return users, nil
}

// UpdateAgent updates an agent owned by uid.
func (r *UserRepo) UpdateAgent(ctx context.Context, agentID, uid, name, avatar, primaryColor, secondaryColor string, wakeMode model.WakeMode) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $1, avatar = $2, primary_color = $3, secondary_color = $4, wake_mode = $5 WHERE id = $6 AND type = $7 AND uid = $8`,
		name, avatar, primaryColor, secondaryColor, wakeMode, agentID, model.UserAgent, uid)
	return err
}

// GetByAPIKey looks up a user by api_key.
func (r *UserRepo) GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT id, type, name, avatar, status, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key FROM users WHERE api_key = $1`, apiKey).
		Scan(&u.ID, &u.Type, &u.Name, &u.Avatar, &u.Status, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

// UpdateAgentAPIKey updates only the api_key for an agent.
func (r *UserRepo) UpdateAgentAPIKey(ctx context.Context, agentID, uid, apiKey string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET api_key = $1 WHERE id = $2 AND type = $3 AND uid = $4`,
		apiKey, agentID, model.UserAgent, uid)
	return err
}

// DeleteAgent deletes an agent owned by uid.
func (r *UserRepo) DeleteAgent(ctx context.Context, agentID, uid string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM users WHERE id = $1 AND type = $2 AND uid = $3`,
		agentID, model.UserAgent, uid)
	return err
}
