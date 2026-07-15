package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

// ConvWebhookDB is the interface satisfied by WebhookRepo.
type ConvWebhookDB interface {
	Create(ctx context.Context, wh *model.ConvWebhook) (*model.ConvWebhook, error)
	GetByID(ctx context.Context, id int64) (*model.ConvWebhook, error)
	GetByAPIKey(ctx context.Context, apiKey string) (*model.ConvWebhook, error)
	ListByConvID(ctx context.Context, convID string) ([]*model.ConvWebhook, error)
	Update(ctx context.Context, wh *model.ConvWebhook) error
	Delete(ctx context.Context, id int64) error
	GetByConvIDAndName(ctx context.Context, convID, name string) (*model.ConvWebhook, error)

	GetAPIKeyHash(ctx context.Context, id int64) (string, error)
	UpdateAPIKeyHash(ctx context.Context, id int64, hash string) error
}

var _ ConvWebhookDB = (*WebhookRepo)(nil)

type WebhookRepo struct {
	pool DBPool
}

func NewWebhookRepo(pool DBPool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

var scanCols = `id, conv_id, name, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, created_by, created_at`

func scanWebhook(row pgx.Row) (*model.ConvWebhook, error) {
	wh := &model.ConvWebhook{}
	var createdAt time.Time
	err := row.Scan(&wh.ID, &wh.ConvID, &wh.Name, &wh.APIKeyPlain, &wh.APIKeyHash,
		&wh.CallbackURL, &wh.Headers, &wh.CIDRWhitelist,
		&wh.CreatedBy, &createdAt)
	if err != nil {
		return nil, err
	}
	wh.CreatedAt = createdAt.UnixMilli()
	return wh, nil
}

func (r *WebhookRepo) Create(ctx context.Context, wh *model.ConvWebhook) (*model.ConvWebhook, error) {
	now := time.Now()
	row := r.pool.QueryRow(ctx,
		`INSERT INTO conv_webhooks (conv_id, name, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING `+scanCols,
		wh.ConvID, wh.Name, wh.APIKeyPlain, wh.APIKeyHash, wh.CallbackURL,
		wh.Headers, wh.CIDRWhitelist, wh.CreatedBy, now)
	return scanWebhook(row)
}

func (r *WebhookRepo) GetByID(ctx context.Context, id int64) (*model.ConvWebhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scanCols+` FROM conv_webhooks WHERE id = $1`, id)
	wh, err := scanWebhook(row)
	if err != nil {
		logger.Error("GetByID query failed", "id", id, "error", err)
		return nil, err
	}
	return wh, nil
}

func (r *WebhookRepo) GetByAPIKey(ctx context.Context, apiKey string) (*model.ConvWebhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scanCols+` FROM conv_webhooks WHERE api_key_plain = $1`, apiKey)
	wh, err := scanWebhook(row)
	if err != nil {
		logger.Error("GetByAPIKey query failed", "error", err)
		return nil, err
	}
	return wh, nil
}

func (r *WebhookRepo) ListByConvID(ctx context.Context, convID string) ([]*model.ConvWebhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT `+scanCols+` FROM conv_webhooks WHERE conv_id = $1 ORDER BY created_at ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.ConvWebhook
	for rows.Next() {
		wh, err := scanWebhook(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, wh)
	}
	return list, nil
}

func (r *WebhookRepo) Update(ctx context.Context, wh *model.ConvWebhook) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE conv_webhooks SET name = $1, callback_url = $2, headers = $3, cidr_whitelist = $4
		 WHERE id = $5`,
		wh.Name, wh.CallbackURL, wh.Headers, wh.CIDRWhitelist, wh.ID)
	if err != nil {
		logger.Error("Update failed", "id", wh.ID, "error", err)
	}
	return err
}

func (r *WebhookRepo) Delete(ctx context.Context, id int64) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM conv_webhooks WHERE id = $1`, id)
	if err != nil {
		logger.Error("Delete failed", "id", id, "error", err)
	}
	return err
}

func (r *WebhookRepo) GetByConvIDAndName(ctx context.Context, convID, name string) (*model.ConvWebhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+scanCols+` FROM conv_webhooks WHERE conv_id = $1 AND name = $2`, convID, name)
	wh, err := scanWebhook(row)
	if err != nil {
		logger.Error("GetByConvIDAndName failed", "conv_id", convID, "name", name, "error", err)
		return nil, err
	}
	return wh, nil
}

func (r *WebhookRepo) GetAPIKeyHash(ctx context.Context, id int64) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT api_key_hash FROM conv_webhooks WHERE id = $1`, id).Scan(&hash)
	return hash, err
}

func (r *WebhookRepo) UpdateAPIKeyHash(ctx context.Context, id int64, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE conv_webhooks SET api_key_hash = $1 WHERE id = $2`, hash, id)
	return err
}

// JSON type helpers for pgx JSONB columns
func (r *WebhookRepo) scanWebhookHeaders(src any) ([]model.WebhookHeader, error) {
	if src == nil {
		return nil, nil
	}
	var headers []model.WebhookHeader
	if data, ok := src.([]byte); ok {
		if err := json.Unmarshal(data, &headers); err != nil {
			return nil, err
		}
	}
	return headers, nil
}

func (r *WebhookRepo) scanStringSlice(src any) ([]string, error) {
	if src == nil {
		return nil, nil
	}
	var s []string
	if data, ok := src.([]byte); ok {
		if err := json.Unmarshal(data, &s); err != nil {
			return nil, err
		}
	}
	return s, nil
}
