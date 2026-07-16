package db

import (
	"context"
	"time"

	"ziziphus/pkg/logger"
	"ziziphus/pkg/model"
)

type UserRepo struct {
	pool DBPool
}

func NewUserRepo(pool DBPool) *UserRepo {
	return &UserRepo{pool: pool}
}

var userAllCols = "id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language"

var userPublicCols = "id, type, name, email, avatar, cover, status, banned, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language"

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO users (id, type, name, email, avatar, cover, status, banned, password, ext_meta, created_at, account, primary_color, secondary_color, uid, wake_mode, api_key, discoverable, allow_direct_chat, headline, language)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21)`,
		u.ID, u.Type, u.Name, u.Email, u.Avatar, u.Cover, u.Status, u.Banned, u.Password, u.ExtMeta, time.UnixMilli(u.CreatedAt), u.Account, u.PrimaryColor, u.SecondaryColor, u.UID, u.WakeMode, u.APIKey, u.Discoverable, u.AllowDirectChat, u.Headline, u.Language)
	return err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT `+userAllCols+` FROM users WHERE id = $1`, id).
		Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &u.Password, &u.ExtMeta, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language)
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
		`SELECT `+userPublicCols+` FROM users WHERE id = ANY($1)`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make(map[string]*model.User, len(ids))
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		result[u.ID] = u
	}
	return result, nil
}

func (r *UserRepo) Search(ctx context.Context, q string, page, size int) ([]*model.User, int, error) {
	count := 0
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE name ILIKE $1 AND NOT banned`, "%"+q+"%").Scan(&count)
	if err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * size
	rows, err := r.pool.Query(ctx,
		`SELECT `+userPublicCols+` FROM users WHERE name ILIKE $1 AND NOT banned ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		"%"+q+"%", size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language); err != nil {
			return nil, 0, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		users = append(users, u)
	}
	return users, count, nil
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT `+userAllCols+` FROM users WHERE email = $1`, email).
		Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &u.Password, &u.ExtMeta, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

func (r *UserRepo) GetByAccount(ctx context.Context, account string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT `+userAllCols+` FROM users WHERE account = $1`, account).
		Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &u.Password, &u.ExtMeta, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language)
	if err != nil {
		return nil, err
	}
	u.CreatedAt = createdAt.UnixMilli()
	return u, nil
}

func (r *UserRepo) Update(ctx context.Context, id, name, avatar, cover, email, primaryColor, secondaryColor, headline string, discoverable, allowDirectChat bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $1, avatar = $2, cover = $3, email = $4, primary_color = $5, secondary_color = $6, discoverable = $7, allow_direct_chat = $8, headline = $9 WHERE id = $10`,
		name, avatar, cover, email, primaryColor, secondaryColor, discoverable, allowDirectChat, headline, id)
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
		`SELECT `+userPublicCols+` FROM users WHERE type = $1 AND uid = $2 ORDER BY created_at ASC`,
		model.UserAgent, uid)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []*model.User
	for rows.Next() {
		u := &model.User{}
		var createdAt time.Time
		if err := rows.Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language); err != nil {
			return nil, err
		}
		u.CreatedAt = createdAt.UnixMilli()
		users = append(users, u)
	}
	return users, nil
}

// UpdateAgent updates an agent owned by uid.
func (r *UserRepo) UpdateAgent(ctx context.Context, agentID, uid, name, avatar, cover, primaryColor, secondaryColor, headline string, wakeMode model.WakeMode, discoverable, allowDirectChat bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET name = $1, avatar = $2, cover = $3, primary_color = $4, secondary_color = $5, wake_mode = $6, discoverable = $7, allow_direct_chat = $8, headline = $9 WHERE id = $10 AND type = $11 AND uid = $12`,
		name, avatar, cover, primaryColor, secondaryColor, wakeMode, discoverable, allowDirectChat, headline, agentID, model.UserAgent, uid)
	return err
}

// GetByAPIKey looks up a user by api_key.
func (r *UserRepo) GetByAPIKey(ctx context.Context, apiKey string) (*model.User, error) {
	u := &model.User{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT `+userPublicCols+` FROM users WHERE api_key = $1`, apiKey).
		Scan(&u.ID, &u.Type, &u.Name, &u.Email, &u.Avatar, &u.Cover, &u.Status, &u.Banned, &createdAt, &u.Account, &u.PrimaryColor, &u.SecondaryColor, &u.UID, &u.WakeMode, &u.APIKey, &u.Discoverable, &u.AllowDirectChat, &u.Headline, &u.Language)
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

// BanUser sets the banned flag on a user and returns the user's ID.
func (r *UserRepo) BanUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET banned = true WHERE id = $1`, userID)
	return err
}

// UnbanUser clears the banned flag on a user.
func (r *UserRepo) UnbanUser(ctx context.Context, userID string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET banned = false WHERE id = $1`, userID)
	return err
}

// UpdateLanguage updates the language preference for a user.
func (r *UserRepo) UpdatePassword(ctx context.Context, userID, password string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password = $1 WHERE id = $2`, password, userID)
	return err
}

func (r *UserRepo) UpdateLanguage(ctx context.Context, userID, language string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET language = $1 WHERE id = $2`, language, userID)
	return err
}

// IsBanned checks whether a user is currently banned.
func (r *UserRepo) IsBanned(ctx context.Context, userID string) (bool, error) {
	var banned bool
	err := r.pool.QueryRow(ctx, `SELECT banned FROM users WHERE id = $1`, userID).Scan(&banned)
	if err != nil {
		return false, err
	}
	return banned, nil
}

// DeleteAccount wipes all user data and deletes the user account.
// Uses a transaction to ensure atomicity.
func (r *UserRepo) DeleteAccount(ctx context.Context, userID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Step through in dependency order — children first, then parents.

	// 1. Remove from all conversations (member records)
	if _, err := tx.Exec(ctx, `DELETE FROM conv_members WHERE user_id = $1`, userID); err != nil {
		return err
	}

	// 2. Anonymize sent messages
	if _, err := tx.Exec(ctx, `UPDATE messages SET body = '' WHERE sender_id = $1`, userID); err != nil {
		return err
	}

	// 3. Remove join requests
	if _, err := tx.Exec(ctx, `DELETE FROM join_requests WHERE user_id = $1`, userID); err != nil {
		return err
	}

	// 4. Remove contacts (both directions)
	if _, err := tx.Exec(ctx, `DELETE FROM contacts WHERE user_id = $1 OR contact_id = $1`, userID); err != nil {
		return err
	}

	// 5. Remove contact requests
	if _, err := tx.Exec(ctx, `DELETE FROM contact_requests WHERE from_user_id = $1 OR to_user_id = $1`, userID); err != nil {
		return err
	}

	// 6. Remove msg receipts (if table exists)
	if _, err := tx.Exec(ctx, `DELETE FROM msg_receipts WHERE user_id = $1`, userID); err != nil {
		logger.Warn("delete msg_receipts failed", "user_id", userID, "error", err)
	}

	// 7. Clear owner references in conversations
	if _, err := tx.Exec(ctx, `UPDATE conversations SET owner_id = '' WHERE owner_id = $1`, userID); err != nil {
		return err
	}

	// 8. Remove sessions
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return err
	}

	// 9. Delete the user itself
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// UpdateBanned updates the banned status for a user (used in unit tests).
func (r *UserRepo) UpdateBanned(ctx context.Context, userID string, banned bool) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET banned = $1 WHERE id = $2`, banned, userID)
	return err
}
