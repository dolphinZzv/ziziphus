package db

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5"
	"siciv.space/agent/panda_ai/pkg/logger"
	"siciv.space/agent/panda_ai/pkg/model"
)

// convWebhookDB is the interface satisfied by WebhookRepo.
// Defined here for use by API handlers and ingest without import cycles.
type ConvWebhookDB interface {
	Create(ctx context.Context, wh *model.ConvWebhook) (*model.ConvWebhook, error)
	GetByID(ctx context.Context, id int64) (*model.ConvWebhook, error)
	GetByToken(ctx context.Context, token string) (*model.ConvWebhook, error)
	ListByConvID(ctx context.Context, convID string) ([]*model.ConvWebhook, error)
	Update(ctx context.Context, wh *model.ConvWebhook) error
	Delete(ctx context.Context, id int64) error
	GetByConvIDAndName(ctx context.Context, convID, name string) (*model.ConvWebhook, error)

	GetAPIKeyHash(ctx context.Context, id int64) (string, error)
	UpdateAPIKeyHash(ctx context.Context, id int64, hash string) error

	InsertWebhookMessage(ctx context.Context, wm *model.WebhookMessage) error
	GetWebhookMessage(ctx context.Context, msgID int64) (*model.WebhookMessage, error)
	ListPendingAudit(ctx context.Context, convID string) ([]*model.WebhookMessage, error)
	UpdateAuditStatus(ctx context.Context, msgID int64, status, actorID string) error
	ListByWebhook(ctx context.Context, whID int64, page, size int) ([]*model.WebhookMessage, int, error)

	InsertAuditLog(ctx context.Context, log *model.WebhookAuditLog) error
	ListAuditLogs(ctx context.Context, convID string, page, size int) ([]*model.WebhookAuditLog, int, error)
}

var _ ConvWebhookDB = (*WebhookRepo)(nil)

type WebhookRepo struct {
	pool DBPool
}

func NewWebhookRepo(pool DBPool) *WebhookRepo {
	return &WebhookRepo{pool: pool}
}

func scanWebhook(row pgx.Row) (*model.ConvWebhook, error) {
	wh := &model.ConvWebhook{}
	var createdAt time.Time
	err := row.Scan(&wh.ID, &wh.ConvID, &wh.Name, &wh.Token, &wh.APIKeyPlain, &wh.APIKeyHash,
		&wh.CallbackURL, &wh.Headers, &wh.CIDRWhitelist, &wh.RequireAudit,
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
		`INSERT INTO conv_webhooks (conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		 RETURNING id, conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at`,
		wh.ConvID, wh.Name, wh.Token, wh.APIKeyPlain, wh.APIKeyHash, wh.CallbackURL,
		wh.Headers, wh.CIDRWhitelist, wh.RequireAudit, wh.CreatedBy, now)
	return scanWebhook(row)
}

func (r *WebhookRepo) GetByID(ctx context.Context, id int64) (*model.ConvWebhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at
		 FROM conv_webhooks WHERE id = $1`, id)
	wh, err := scanWebhook(row)
	if err != nil {
		logger.Error("GetByID query failed", "id", id, "error", err)
		return nil, err
	}
	return wh, nil
}

func (r *WebhookRepo) GetByToken(ctx context.Context, token string) (*model.ConvWebhook, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT id, conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at
		 FROM conv_webhooks WHERE token = $1`, token)
	wh, err := scanWebhook(row)
	if err != nil {
		logger.Error("GetByToken query failed", "token", token, "error", err)
		return nil, err
	}
	return wh, nil
}

func (r *WebhookRepo) ListByConvID(ctx context.Context, convID string) ([]*model.ConvWebhook, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at
		 FROM conv_webhooks WHERE conv_id = $1 ORDER BY created_at ASC`, convID)
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
		`UPDATE conv_webhooks SET name = $1, callback_url = $2, headers = $3, cidr_whitelist = $4, require_audit = $5
		 WHERE id = $6`,
		wh.Name, wh.CallbackURL, wh.Headers, wh.CIDRWhitelist, wh.RequireAudit, wh.ID)
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
		`SELECT id, conv_id, name, token, api_key_plain, api_key_hash, callback_url, headers, cidr_whitelist, require_audit, created_by, created_at
		 FROM conv_webhooks WHERE conv_id = $1 AND name = $2`, convID, name)
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

func (r *WebhookRepo) InsertWebhookMessage(ctx context.Context, wm *model.WebhookMessage) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhook_messages (msg_id, webhook_id, conv_id, audit_status, source_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		wm.MsgID, wm.WebhookID, wm.ConvID, wm.AuditStatus, wm.SourceIP, time.UnixMilli(wm.CreatedAt))
	return err
}

func (r *WebhookRepo) GetWebhookMessage(ctx context.Context, msgID int64) (*model.WebhookMessage, error) {
	wm := &model.WebhookMessage{}
	var createdAt time.Time
	err := r.pool.QueryRow(ctx,
		`SELECT msg_id, webhook_id, conv_id, audit_status, source_ip, created_at
		 FROM webhook_messages WHERE msg_id = $1`, msgID).
		Scan(&wm.MsgID, &wm.WebhookID, &wm.ConvID, &wm.AuditStatus, &wm.SourceIP, &createdAt)
	if err != nil {
		return nil, err
	}
	wm.CreatedAt = createdAt.UnixMilli()
	return wm, nil
}

func (r *WebhookRepo) ListPendingAudit(ctx context.Context, convID string) ([]*model.WebhookMessage, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT msg_id, webhook_id, conv_id, audit_status, source_ip, created_at
		 FROM webhook_messages WHERE conv_id = $1 AND audit_status = 'pending'
		 ORDER BY created_at ASC`, convID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*model.WebhookMessage
	for rows.Next() {
		wm := &model.WebhookMessage{}
		var createdAt time.Time
		if err := rows.Scan(&wm.MsgID, &wm.WebhookID, &wm.ConvID, &wm.AuditStatus, &wm.SourceIP, &createdAt); err != nil {
			return nil, err
		}
		wm.CreatedAt = createdAt.UnixMilli()
		list = append(list, wm)
	}
	return list, nil
}

func (r *WebhookRepo) UpdateAuditStatus(ctx context.Context, msgID int64, status, actorID string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE webhook_messages SET audit_status = $1 WHERE msg_id = $2`, status, msgID)
	return err
}

func (r *WebhookRepo) ListByWebhook(ctx context.Context, whID int64, page, size int) ([]*model.WebhookMessage, int, error) {
	offset := (page - 1) * size
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM webhook_messages WHERE webhook_id = $1`, whID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT msg_id, webhook_id, conv_id, audit_status, source_ip, created_at
		 FROM webhook_messages WHERE webhook_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		whID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*model.WebhookMessage
	for rows.Next() {
		wm := &model.WebhookMessage{}
		var createdAt time.Time
		if err := rows.Scan(&wm.MsgID, &wm.WebhookID, &wm.ConvID, &wm.AuditStatus, &wm.SourceIP, &createdAt); err != nil {
			return nil, 0, err
		}
		wm.CreatedAt = createdAt.UnixMilli()
		list = append(list, wm)
	}
	return list, total, nil
}

func (r *WebhookRepo) InsertAuditLog(ctx context.Context, log *model.WebhookAuditLog) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO webhook_audit_logs (webhook_id, conv_id, msg_id, action, actor_id, reason, caller_ip, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		log.WebhookID, log.ConvID, log.MsgID, log.Action, log.ActorID,
		log.Reason, log.CallerIP, time.UnixMilli(log.CreatedAt))
	return err
}

func (r *WebhookRepo) ListAuditLogs(ctx context.Context, convID string, page, size int) ([]*model.WebhookAuditLog, int, error) {
	offset := (page - 1) * size
	var total int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM webhook_audit_logs WHERE conv_id = $1`, convID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, webhook_id, conv_id, msg_id, action, actor_id, COALESCE(reason, ''), COALESCE(caller_ip, ''), created_at
		 FROM webhook_audit_logs WHERE conv_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
		convID, size, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var list []*model.WebhookAuditLog
	for rows.Next() {
		l := &model.WebhookAuditLog{}
		var createdAt time.Time
		if err := rows.Scan(&l.ID, &l.WebhookID, &l.ConvID, &l.MsgID, &l.Action, &l.ActorID, &l.Reason, &l.CallerIP, &createdAt); err != nil {
			return nil, 0, err
		}
		l.CreatedAt = createdAt.UnixMilli()
		list = append(list, l)
	}
	return list, total, nil
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
